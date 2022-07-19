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

	"github.com/kendalharland/mktree/parse"
)

const (
	defaultDirMode  = os.FileMode(0777) | os.ModeDir
	defaultFileMode = os.FileMode(0666)
)

func defaultRootDir(name string) *Dir {
	return &Dir{
		Name:  name,
		Perms: defaultDirMode,
	}
}

func Interpret(r io.Reader) (*Tree, error) {
	return (&Interpreter{}).Interpret(r)
}

type Interpreter struct {
	Root   string
	Vars   map[string]string
	Stderr io.Writer

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
	return i.Exec(bytes.NewReader(input), opts...)
}

func (i *Interpreter) Exec(r io.Reader, opts ...Option) error {
	t, err := i.Interpret(r)
	if err != nil {
		return err
	}
	// append builtins first so the user can override them.
	opts = append(builtins(i), opts...)
	return createTree(t, opts...)
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

func createTree(t *Tree, opts ...Option) error {
	ctx := &treeContext{}
	for _, o := range opts {
		o.apply(ctx)
	}
	return t.createDir(ctx, t.root)
}

func evalConfig(c *parse.Config, root *Dir) error {
	for _, e := range c.SExprs {
		if err := evalDirChild(root, e); err != nil {
			return err
		}
	}
	return nil
}

func evalDirChild(parent *Dir, e *parse.SExpr) (err error) {
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

func evalFileChild(parent *File, e *parse.SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case parse.AttributeTokenKind:
		err = evalAttr(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalDir(parent *Dir, e *parse.SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a directory name")
	}

	name, err := evalString(e.Args[0])
	if err != nil {
		return err
	}

	d := &Dir{Name: filepath.Join(parent.Name, name)}
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
	case *File:
		return t.setAttribute(name, args)
	case *Dir:
		return t.setAttribute(name, args)
	default:
		panic(fmt.Errorf("setAttr(%q) called on owner of type `%T` which does not have attributes", name, t))
	}
}

func evalFile(parent *Dir, e *parse.SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a filename")
	}

	name, err := evalString(e.Args[0])
	if err != nil {
		return err
	}

	name = filepath.Join(parent.Name, name)
	name = filepath.Clean(name)
	f := &File{
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

var ErrInterpret = errors.New("interpet error")

func interpretError(format string, args ...interface{}) error {
	return parse.Errorf(ErrInterpret, format, args...)
}
