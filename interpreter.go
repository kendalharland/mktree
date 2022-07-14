package mktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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

func Interpret(r io.Reader) (*Dir, error) {
	i := &Interpreter{}
	return i.Interpret(r)
}

type Interpreter struct {
	Root   string
	Vars   map[string]string
	Stderr io.Writer
}

func (i *Interpreter) init() error {
	if i.Vars == nil {
		i.Vars = make(map[string]string)
	}
	return nil
}

func (i *Interpreter) Interpret(r io.Reader) (*Dir, error) {
	i.init()

	source, err := preprocess(r, i.Vars)
	if err != nil {
		return nil, err
	}

	p := &Parser{Stderr: i.Stderr}
	config, err := p.Parse(source)
	if err != nil {
		return nil, err
	}

	root := defaultRootDir(i.Root)
	if err := evalConfig(config, root); err != nil {
		return nil, err
	}

	return root, nil
}

func evalConfig(c *Config, root *Dir) error {
	for _, e := range c.SExprs {
		if err := evalDirChild(root, e); err != nil {
			return err
		}
	}
	return nil
}

func evalDirChild(parent *Dir, e *SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case AttributeTokenKind:
		err = evalAttr(parent, e)
	case DirTokenKind:
		err = evalDir(parent, e)
	case FileTokenKind:
		err = evalFile(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalFileChild(parent *File, e *SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case AttributeTokenKind:
		err = evalAttr(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalDir(parent *Dir, e *SExpr) error {
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

func evalAttr(owner interface{}, e *SExpr) error {
	attr, err := evalAttrName(e.Literal)
	if err != nil {
		return err
	}
	return setAttr(owner, attr, e.Args)
}

func setAttr(owner interface{}, name string, args []*Arg) error {
	switch t := owner.(type) {
	case *File:
		return t.setAttribute(name, args)
	case *Dir:
		return t.setAttribute(name, args)
	default:
		panic(fmt.Errorf("setAttr(%q) called on owner of type `%T` which does not have attributes", name, t))
	}
}

func evalFile(parent *Dir, e *SExpr) error {
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

func evalAttrName(l *Literal) (string, error) {
	if l.Token.Kind != AttributeTokenKind {
		return "", interpretError("%q is not an attribute", l.Token.Value)
	}
	// Skip leading '@'.
	return l.Token.Value[1:], nil
}

func evalString(a *Arg) (string, error) {
	l := a.Literal
	if l == nil || l.Token.Kind != StringTokenKind {
		return "", interpretError("%v is not a string", a.Token)
	}
	return l.Token.Value, nil
}

func evalFileMode(a *Arg) (os.FileMode, error) {
	l := a.Literal

	if l.Token.Kind != NumberTokenKind {
		return 0, interpretError("%q is not a number", l.Token.Value)
	}
	n, err := strconv.ParseUint(l.Token.Value, 8, 32)
	if err != nil {
		return 0, interpretError("%q is not a file mode octal", l.Token.Value)
	}

	return os.FileMode(uint32(n)), nil
}

func interpretError(format string, args ...interface{}) error {
	return errorf(ErrInterpret, format, args...)
}
