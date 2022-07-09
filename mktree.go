package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultDirPerms  = os.FileMode(0666)
	DefaultFilePerms = os.FileMode(0755)
)

type Dir struct {
	Name  string
	Perms os.FileMode
	Files []*File
	Dirs  []*Dir
}

func (d *Dir) SetPerms(m os.FileMode) {
	d.Perms = m | fs.ModeDir
}

type File struct {
	Name         string
	Perms        os.FileMode
	TemplatePath string
	TemplateArgs []string
}

func (f *File) SetPerms(m os.FileMode) {
	f.Perms = m
}

func Execute(r io.Reader, dir string) error {
	p := &Parser{}
	c, err := p.Parse(r)
	if err != nil {
		return err
	}

	root, err := evalConfig(c)
	if err != nil {
		return err
	}

	printDir(root)

	return err
}

func evalConfig(c *Config) (*Dir, error) {
	var root Dir
	for _, e := range c.SExprs {
		if err := evalSExprAsDirChild(&root, e); err != nil {
			return nil, err
		}
	}
	return &root, nil
}

// TODO: Merge with file version and check attr in setters.
func evalSExprAsDirChild(parent *Dir, e *SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case AttributeTokenKind:
		err = evalSExprAsDirAttr(parent, e)
	case DirTokenKind:
		err = evalSExprAsDir(parent, e)
	case FileTokenKind:
		err = evalSExprAsFile(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalSExprAsFileChild(parent *File, e *SExpr) (err error) {
	switch e.Literal.Token.Kind {
	case AttributeTokenKind:
		err = evalSExprAsFileAttr(parent, e)
	default:
		err = interpretError("invalid s-expression: %v", e.Literal.Token)
	}
	return err
}

func evalSExprAsDir(parent *Dir, e *SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a directory name")
	}

	name, err := evalString(e.Args[0].Literal)
	if err != nil {
		return err
	}

	d := &Dir{Name: filepath.Join(parent.Name, name)}
	for _, arg := range e.Args[1:] {
		if err := evalSExprAsDirChild(d, arg.SExpr); err != nil {
			return err
		}
	}
	if d.Perms == os.FileMode(0) {
		d.SetPerms(DefaultDirPerms)
	}

	parent.Dirs = append(parent.Dirs, d)
	return nil
}

func evalSExprAsDirAttr(parent *Dir, e *SExpr) error {
	attr, err := evalAttr(e.Literal)
	if err != nil {
		return err
	}

	if attr == "perms" {
		value, err := evalNumber(e.Args[0].Literal)
		if err != nil {
			return err
		}
		parent.SetPerms(os.FileMode(value))
		return nil
	}

	return interpretError("%q is not a valid directory attribute", attr)
}

func evalSExprAsFileAttr(parent *File, e *SExpr) error {
	attr, err := evalAttr(e.Literal)
	if err != nil {
		return err
	}

	if attr == "perms" {
		value, err := evalNumber(e.Args[0].Literal)
		if err != nil {
			return err
		}
		parent.SetPerms(os.FileMode(value))
		return nil
	}

	return interpretError("%q is not a valid directory attribute", attr)
}

func evalSExprAsFile(parent *Dir, e *SExpr) error {
	if len(e.Args) < 1 {
		return interpretError("expected a directory name")
	}

	name, err := evalString(e.Args[0].Literal)
	if err != nil {
		return err
	}

	f := &File{Name: filepath.Join(parent.Name, name)}
	for _, arg := range e.Args[1:] {
		if err := evalSExprAsFileChild(f, arg.SExpr); err != nil {
			return err
		}
	}
	if f.Perms == os.FileMode(0) {
		f.SetPerms(DefaultFilePerms)
	}

	parent.Files = append(parent.Files, f)
	return nil
}

func evalAttr(l *Literal) (string, error) {
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

func printDir(root *Dir) {
	fmt.Printf("%v %v\n", root.Perms, root.Name)
	for _, d := range root.Dirs {
		printDir(d)
	}
	for _, f := range root.Files {
		printFile(f)
	}
}

func printFile(f *File) {
	fmt.Printf("%v %v\n", f.Perms, f.Name)
}
