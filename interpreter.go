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

func createTree(t *Tree, sourceRoot string, opts ...Option) error {
	ctx := &thread{sourceRoot: sourceRoot}
	for _, o := range opts {
		o.apply(ctx)
	}
	return createDir(ctx, t.root)
}

func defaultRootDir(name string) *dir {
	return &dir{
		Name:  name,
		Perms: defaultDirMode,
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

func evalDirChild(parent *dir, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalAttr(parent, e)
	case parse.DirTokenKind:
		err = evalDir(parent, e)
	case parse.FileTokenKind:
		err = evalFile(parent, e)
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

func evalDir(parent *dir, e *parse.SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a directory name")
	}

	name, err := evalString(e.Args[0])
	if err != nil {
		return err
	}

	d := &dir{Name: filepath.Join(parent.Name, name)}
	for _, arg := range e.Args[1:] {
		if err := evalDirChild(d, arg.SExpr); err != nil {
			return err
		}
	}
	if d.Perms == os.FileMode(0) {
		d.Perms = defaultDirMode
	}

	parent.addDir(d)
	return nil
}

func evalAttr(owner interface{}, e *parse.SExpr) error {
	attr, err := evalAttrName(e.Literal)
	if err != nil {
		return err
	}
	return setAttr(owner, attr, e.Args)
}

func setAttr(owner interface{}, name string, args []*parse.Arg) error {
	switch t := owner.(type) {
	case *file:
		return t.setAttribute(name, args)
	case *dir:
		return t.setAttribute(name, args)
	default:
		panic(fmt.Errorf("setAttr(%q) called on owner of type `%T` which does not have attributes", name, t))
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

	name = filepath.Join(parent.Name, name)
	name = filepath.Clean(name)
	f := &file{
		Name:  name,
		Perms: defaultFileMode,
	}
	for _, arg := range e.Args[1:] {
		if err := evalFileChild(f, arg.SExpr); err != nil {
			return err
		}
	}

	parent.addFile(f)
	return nil
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

func createDir(ctx *thread, d *dir) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	if err := os.MkdirAll(d.Name, d.Perms); err != nil {
		return err
	}
	for _, child := range d.Dirs {
		if err := createDir(ctx, child); err != nil {
			return err
		}
	}
	for _, child := range d.Files {
		if err := createFile(ctx, child); err != nil {
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
	if err := os.MkdirAll(filepath.Dir(f.Name), defaultDirMode); err != nil {
		return err
	}
	return ioutil.WriteFile(f.Name, []byte(contents), f.Perms)
}

func fileContents(ctx *thread, f *file) (string, error) {
	if len(f.Contents) > 0 {
		return string(f.Contents), nil
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

// Errors.

func interpretError(format string, args ...interface{}) error {
	return parse.Errorf(ErrInterpret, format, args...)
}
