package mktree

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultDirMode  = os.FileMode(0777)
	defaultFileMode = os.FileMode(0755)
)

func Interpret(r io.Reader) (*Dir, error) {
	i := &Interpreter{}
	return i.Interpret(r)
}

type Interpreter struct {
	Root string
	Vars map[string]string
}

func (i *Interpreter) init() error {
	if i.Vars == nil {
		i.Vars = make(map[string]string)
	}
	if i.Root == "" {
		root, err := os.Getwd()
		if err != nil {
			return errors.New("unable to choose the root directory. Please specify one when creating the interpreter")
		}
		i.Root = root
	}
	return nil
}

func (i *Interpreter) Interpret(r io.Reader) (*Dir, error) {
	i.init()

	source, err := preprocess(r, i.Vars)
	if err != nil {
		return nil, err
	}

	config, err := Parse(source)
	if err != nil {
		return nil, err
	}

	root := &Dir{
		Name:  i.Root,
		Perms: defaultDirMode | os.ModeDir,
	}
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

// TODO: Merge with file version and check attr in setters.
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

	name, err := evalString(e.Args[0].Literal)
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
		d.setPerms(defaultDirMode)
	}

	parent.Dirs = append(parent.Dirs, d)
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
		t.setAttribute(name, args)
	case *Dir:
		t.setAttribute(name, args)
	default:
		panic(fmt.Errorf("setAttr(%q) called on owner of type `%T` which does not have attributes", name, t))
	}
	return nil
}

func evalFile(parent *Dir, e *SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a directory name")
	}

	name, err := evalString(e.Args[0].Literal)
	if err != nil {
		return err
	}

	f := &File{
		Name:  filepath.Join(parent.Name, name),
		Perms: defaultFileMode,
	}
	for _, arg := range e.Args[1:] {
		if err := evalFileChild(f, arg.SExpr); err != nil {
			return err
		}
	}

	parent.Files = append(parent.Files, f)
	return nil
}

func evalAttrName(l *Literal) (string, error) {
	if l.Token.Kind != AttributeTokenKind {
		return "", interpretError("%q is not an attribute", l.Token.Value)
	}
	// Skip leading '@'.
	return l.Token.Value[1:], nil
}

func evalString(l *Literal) (string, error) {
	if l.Token.Kind != StringTokenKind {
		return "", interpretError("%q is not a string", l.Token.Value)
	}
	return l.Token.Value, nil
}

func evalNumber(l *Literal) (float64, error) {
	if l.Token.Kind != NumberTokenKind {
		return 0, interpretError("%q is not a number", l.Token.Value)
	}
	n, err := strconv.ParseFloat(l.Token.Value, 64)
	if err != nil {
		return 0, interpretError("%q is not a number", l.Token.Value)
	}
	return n, nil
}

func interpretError(format string, args ...interface{}) error {
	return fmt.Errorf("interpret error: "+format, args...)
}
