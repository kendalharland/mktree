package mktree

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/kendalharland/mktree/parse"
)

type Tree struct {
	root *dir
}

func (t *Tree) DebugPrint(w io.Writer) {
	t.root.debugPrint(w)
}

type dir struct {
	Name  string
	Perms os.FileMode
	Files []*file
	Dirs  []*dir
}

func (d *dir) debugPrint(w io.Writer) {
	fmt.Fprintf(w, "%v %v\n", d.Perms, d.Name)
	for _, child := range d.Dirs {
		child.debugPrint(w)
	}
	for _, child := range d.Files {
		child.debugPrint(w)
	}
}

func (d *dir) addDir(child *dir) {
	d.Dirs = append(d.Dirs, child)
}

func (d *dir) addFile(child *file) {
	d.Files = append(d.Files, child)
}

func (d *dir) setAttribute(name string, args []*parse.Arg) error {
	switch name {
	case "perms":
		return d.setPerms(args)
	}
	return interpretError("invalid file attribute %q", name)
}

func (d *dir) setPerms(args []*parse.Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	d.Perms = mode | fs.ModeDir
	return nil
}

type file struct {
	Name             string
	Perms            os.FileMode
	Contents         []byte
	TemplateFilename string
}

func (f *file) debugPrint(w io.Writer) {
	fmt.Fprintf(w, "%v %v %d\n", f.Perms, f.Name, len(f.Contents))
}

func (f *file) setAttribute(name string, args []*parse.Arg) error {
	switch name {
	case "perms":
		return f.setPerms(args)
	case "template":
		return f.setTemplate(args)
	case "contents":
		return f.setContents(args)
	}
	return interpretError("invalid file attribute %q", name)
}

func (f *file) setPerms(args []*parse.Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	f.Perms = mode
	return nil
}

func (f *file) setTemplate(args []*parse.Arg) error {
	if len(f.Contents) > 0 {
		return errors.New("cannot set @template if @contents is set")
	}
	filename, err := evalString(args[0])
	if err != nil {
		return err
	}
	f.TemplateFilename = filename
	return nil
}

func (f *file) setContents(args []*parse.Arg) error {
	if f.TemplateFilename != "" {
		return errors.New("cannot set @contents if @template is set")
	}
	contents, err := evalString(args[0])
	if err != nil {
		return err
	}
	f.Contents = []byte(contents)
	return nil
}
