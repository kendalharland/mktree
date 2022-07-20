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

var errInterpret = errors.New("interpet error")

const (
	defaultDirMode  = os.FileMode(0777) | os.ModeDir
	defaultFileMode = os.FileMode(0666)
)

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

// ExecFile interprets and executes the given file.
// If r is nil, the file is read and executed. Otherwise, the input is read from r
// and the filename is used only to add context to error messages.
// If r is nil and the filename is empty, an error is returned.
func (i *Interpreter) ExecFile(r io.Reader, filename string, opts ...Option) error {
	tree, err := i.InterpretFile(r, filename)
	if err != nil {
		return err
	}

	// append builtin options first so the user can override them.
	opts = append(builtins(i), opts...)
	t := newThread(filename, opts...)
	return createTree(t, tree)
}

// InterpretFile interprets the given file.
// If r is nil, the file is read and interpreted. Otherwise, the input is read from r
// and the filename is used only to add context to error messages.
// If r is nil and the filename is empty, an error is returned.
func (i *Interpreter) InterpretFile(r io.Reader, filename string) (*Tree, error) {
	i.init()

	if r == nil {
		if filename == "" {
			return nil, errors.New("the caller must provide a non-nil reader or the path to a file")
		}
		source, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(source)
	}

	source, err := preprocess(r, i.Vars, i.AllowUndefinedVars)
	if err != nil {
		return nil, err
	}

	p := &parse.Parser{Stderr: i.Stderr}
	tree, err := p.Parse(source)
	if err != nil {
		return nil, err
	}

	root := defaultRootDir(i.Root)
	if err := evalTree(tree, root); err != nil {
		return nil, err
	}

	return &Tree{root}, nil
}

func defaultRootDir(name string) *dir {
	return &dir{
		name:  name,
		perms: defaultDirMode,
	}
}

func evalTree(t *parse.Tree, root *dir) error {
	for _, e := range t.SExprs {
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

	name, err := evalRelPath(parent, e.Args[0])
	if err != nil {
		return err
	}

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

func evalDirAttr(d *dir, e *parse.SExpr) error {
	attr, err := evalAttrName(e.Literal)
	if err != nil {
		return err
	}
	switch attr {
	case "perms":
		return d.setPerms(e.Args)
	}
	return interpretError("invalid dir attribute %q", attr)
}

func evalDirChild(parent *dir, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalDirAttr(parent, e)
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

func evalFile(parent *dir, e *parse.SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a filename")
	}

	name, err := evalRelPath(parent, e.Args[0])
	if err != nil {
		return err
	}

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

func evalFileAttr(f *file, e *parse.SExpr) error {
	attr, err := evalAttrName(e.Literal)
	if err != nil {
		return err
	}
	switch attr {
	case "perms":
		return evalFilePerms(f, e.Args)
	case "template":
		return evalFileTemplate(f, e.Args)
	case "contents":
		return evalFileContents(f, e.Args)
	}
	return interpretError("invalid file attribute %q", attr)
}

func evalFileChild(parent *file, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalFileAttr(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalFileContents(f *file, args []*parse.Arg) error {
	if f.templatePath != "" {
		return errors.New("cannot set @contents if @template is set")
	}
	contents, err := evalString(args[0])
	if err != nil {
		return err
	}
	f.setContents([]byte(contents))
	return nil
}

func evalFilePerms(f *file, args []*parse.Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	f.setPerms(mode)
	return nil
}

func evalFileTemplate(f *file, args []*parse.Arg) error {
	if len(f.contents) > 0 {
		return errors.New("cannot set @template if @contents is set")
	}
	filename, err := evalString(args[0])
	if err != nil {
		return err
	}
	f.setTemplate(filename)
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

	name, err := evalRelPath(parent, e.Args[1])
	if err != nil {
		return err
	}

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

func evalRelPath(parent *dir, a *parse.Arg) (string, error) {
	name, err := evalString(a)
	if err != nil {
		return "", err
	}

	name = filepath.Join(parent.name, name)
	return name, nil
}

func evalLinkAttr(l *link, e *parse.SExpr) error {
	attr, err := evalAttrName(e.Literal)
	if err != nil {
		return err
	}
	switch attr {
	case "symbolic":
		l.symbolic = true
		return nil
	}
	return interpretError("invalid link attribute %q", attr)
}

func evalLinkChild(parent *link, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalLinkAttr(parent, e)
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

func createTree(thr *thread, t *Tree) error {
	return createDir(thr, t.root)
}

func createDir(thr *thread, d *dir) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	if err := os.MkdirAll(d.name, d.perms); err != nil {
		return err
	}
	for _, child := range d.dirs {
		if err := createDir(thr, child); err != nil {
			return err
		}
	}
	for _, child := range d.files {
		if err := createFile(thr, child); err != nil {
			return err
		}
	}
	for _, child := range d.links {
		if err := createLink(thr, child); err != nil {
			return err
		}
	}

	return nil
}

func createFile(thr *thread, f *file) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	contents, err := fileContents(thr, f)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.name), defaultDirMode); err != nil {
		return err
	}
	return ioutil.WriteFile(f.name, []byte(contents), f.perms)
}

func createLink(thr *thread, l *link) error {
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

func fileContents(thr *thread, f *file) (string, error) {
	if len(f.contents) > 0 {
		return string(f.contents), nil
	}
	if len(f.templatePath) > 0 {
		filename := filepath.Join(thr.sourceRoot, f.templatePath)
		return execTemplateFile(filename, thr.templateFuncs)
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
	return parse.Errorf(errInterpret, format, args...)
}
