package mktree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/kendalharland/mktree/parse"
	"golang.org/x/sys/unix"
)

var ErrInterpret = errors.New("interpet error")

const (
	defaultDirMode  = os.FileMode(0777) | os.ModeDir
	defaultFileMode = os.FileMode(0666)
)

func Interpret(r io.Reader) (*Tree, error) {
	return (&Interpreter{}).Interpret(r)
}

type Interpreter struct {
	Root               string
	Vars               map[string]string
	Stderr             io.Writer
	AllowUndefinedVars bool
}

func (i *Interpreter) init() error {
	if _, ok := i.Vars["root_dir"]; ok {
		return fmt.Errorf("cannot set variable 'root_dir'")
	}
	if i.Root == "" {
		i.Root = "."
	}
	if i.Vars == nil {
		i.Vars = map[string]string{}
	}
	i.Vars["root_dir"] = i.Root
	return nil
}

func (i *Interpreter) ExecFile(filename string, opts ...Option) error {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return i.Exec(bytes.NewReader(input), filename, opts...)
}

func (i *Interpreter) Exec(r io.Reader, filename string, opts ...Option) error {
	t, err := i.Interpret(r)
	if err != nil {
		return err
	}
	// append builtins first so the user can override them.
	opts = append(builtins(i), opts...)
	sourceRoot := filepath.Dir(filename)
	if sourceRoot == "" {
		sourceRoot = "."
	}
	return createTree(t, sourceRoot, opts...)
}

func (i *Interpreter) Interpret(r io.Reader) (*Tree, error) {
	i.init()

	source, err := preprocess(r, i.Vars, i.AllowUndefinedVars)
	if err != nil {
		return nil, err
	}

	p := &parse.Parser{Stderr: i.Stderr}
	config, err := p.Parse(source)
	if err != nil {
		return nil, err
	}

	root := defaultRootDir(i.Root)
	if err := evalConfig(config, root); err != nil {
		return nil, err
	}

	return &Tree{root}, nil
}

func (i *Interpreter) InterpretFile(filename string) (*Tree, error) {
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return i.Interpret(bytes.NewReader(input))
}

func defaultRootDir(name string) *dir {
	return &dir{
		name:  name,
		perms: defaultDirMode,
	}
}

func evalConfig(c *parse.Config, root *dir) error {
	for _, e := range c.SExprs {
		if err := evalDirChild(root, e); err != nil {
			return err
		}
	}
	return nil
}

func evalDir(parent *dir, e *parse.SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a directory name")
	}

	name, err := evalString(e.Args[0])
	if err != nil {
		return err
	}

	name = filepath.Join(parent.name, name)
	name = filepath.Clean(name)
	d := &dir{
		name:  name,
		perms: defaultDirMode,
	}
	for _, arg := range e.Args[1:] {
		if err := evalDirChild(d, arg.SExpr); err != nil {
			return err
		}
	}

	parent.addDir(d)
	return nil
}

func evalAttr(owner interface{}, e *parse.SExpr) error {
	attr, err := evalAttrName(e.Literal)
	if err != nil {
		return err
	}
	args := e.Args
	switch t := owner.(type) {
	case *dir:
		return t.setAttribute(attr, args)
	case *file:
		return t.setAttribute(attr, args)
	case *link:
		return t.setAttribute(attr, args)
	default:
		panic(fmt.Errorf("evalAttr(%q) called on owner of type `%T` which does not have attributes", attr, t))
	}
}

func evalFile(parent *dir, e *parse.SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a filename")
	}

	name, err := evalString(e.Args[0])
	if err != nil {
		return err
	}

	name = filepath.Join(parent.name, name)
	name = filepath.Clean(name)
	f := &file{
		name:  name,
		perms: defaultFileMode,
	}
	for _, arg := range e.Args[1:] {
		if err := evalFileChild(f, arg.SExpr); err != nil {
			return err
		}
	}

	parent.addFile(f)
	return nil
}

func evalLink(parent *dir, e *parse.SExpr) error {
	if len(e.Args) < 2 {
		return interpretError("expected a link target and name")
	}

	target, err := evalString(e.Args[0])
	if err != nil {
		return err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(parent.name, target)
	}

	name, err := evalString(e.Args[1])
	if err != nil {
		return err
	}
	// TODO: Put this in a helper.
	name = filepath.Join(parent.name, name)

	l := &link{
		name:   name,
		target: target,
	}
	for _, arg := range e.Args[2:] {
		if err := evalLinkChild(l, arg.SExpr); err != nil {
			return err
		}
	}

	parent.addLink(l)
	return nil

}

func evalDirChild(parent *dir, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalAttr(parent, e)
	case parse.DirTokenKind:
		err = evalDir(parent, e)
	case parse.FileTokenKind:
		err = evalFile(parent, e)
	case parse.LinkTokenKind:
		err = evalLink(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalFileChild(parent *file, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalAttr(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalLinkChild(parent *link, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalAttr(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token.Kind)
	}
	return err
}

func evalAttrName(l *parse.Literal) (string, error) {
	if l.Token.Kind != parse.AttributeTokenKind {
		return "", interpretError("%q is not an attribute", l.Token.Value)
	}
	// Skip leading '@'.
	return l.Token.Value[1:], nil
}

func evalString(a *parse.Arg) (string, error) {
	l := a.Literal
	if l == nil || l.Token.Kind != parse.StringTokenKind {
		return "", interpretError("%v is not a string", a.Token)
	}
	return l.Token.Value, nil
}

func evalFileMode(a *parse.Arg) (os.FileMode, error) {
	l := a.Literal

	if l.Token.Kind != parse.NumberTokenKind || len(l.Token.Value) != 4 {
		return 0, interpretError("invalid file mode %q", l.Token.Value)
	}
	n, err := strconv.ParseUint(l.Token.Value, 8, 32)
	if err != nil {
		return 0, interpretError("invalid file mode %q", l.Token.Value)
	}

	return os.FileMode(uint32(n)), nil
}

//
// Generators
//

func createTree(t *Tree, sourceRoot string, opts ...Option) error {
	ctx := &thread{sourceRoot: sourceRoot}
	for _, o := range opts {
		o.apply(ctx)
	}
	return createDir(ctx, t.root)
}

func createDir(ctx *thread, d *dir) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	if err := os.MkdirAll(d.name, d.perms); err != nil {
		return err
	}
	for _, child := range d.dirs {
		if err := createDir(ctx, child); err != nil {
			return err
		}
	}
	for _, child := range d.files {
		if err := createFile(ctx, child); err != nil {
			return err
		}
	}
	for _, child := range d.links {
		if err := createLink(ctx, child); err != nil {
			return err
		}
	}

	return nil
}

func createFile(ctx *thread, f *file) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	contents, err := fileContents(ctx, f)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.name), defaultDirMode); err != nil {
		return err
	}
	return ioutil.WriteFile(f.name, []byte(contents), f.perms)
}

func fileContents(ctx *thread, f *file) (string, error) {
	if len(f.contents) > 0 {
		return string(f.contents), nil
	}
	if len(f.TemplateFilename) > 0 {
		filename := filepath.Join(ctx.sourceRoot, f.TemplateFilename)
		return execTemplateFile(filename, ctx.templateFuncs)
	}
	return "", nil
}

func execTemplateFile(filename string, funcMap template.FuncMap) (string, error) {
	name := filepath.Base(filename)
	tmpl, err := template.New(name).Funcs(funcMap).ParseFiles(filename)
	if err != nil {
		return "", err
	}
	var contents bytes.Buffer
	if err := tmpl.Execute(&contents, nil); err != nil {
		return "", err
	}
	return contents.String(), nil
}

func createLink(ctx *thread, l *link) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	if err := os.MkdirAll(filepath.Dir(l.name), defaultDirMode); err != nil {
		return err
	}

	linker := os.Link
	if l.symbolic {
		linker = os.Symlink
	}

	return linker(l.target, l.name)
}

// Errors.

func interpretError(format string, args ...interface{}) error {
	return parse.Errorf(ErrInterpret, format, args...)
}
