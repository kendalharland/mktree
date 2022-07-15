package mktree

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/kendalharland/mktree/parse"
)

type Dir struct {
	Name  string
	Perms os.FileMode
	Files []*File
	Dirs  []*Dir
}

func (d *Dir) DebugPrint(w io.Writer) {
	fmt.Fprintf(w, "%v %v\n", d.Perms, d.Name)
	for _, child := range d.Dirs {
		child.DebugPrint(w)
	}
	for _, child := range d.Files {
		child.DebugPrint(w)
	}
}

func (d *Dir) addDir(child *Dir) {
	d.Dirs = append(d.Dirs, child)
}

func (d *Dir) addFile(child *File) {
	d.Files = append(d.Files, child)
}

func (d *Dir) setAttribute(name string, args []*parse.Arg) error {
	switch name {
	case "perms":
		return d.setPerms(args)
	}
	return interpretError("invalid file attribute %q", name)
}

func (d *Dir) setPerms(args []*parse.Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	d.Perms = mode | fs.ModeDir
	return nil
}

type File struct {
	Name             string
	Perms            os.FileMode
	Contents         []byte
	TemplateFilename string
}

func (f *File) DebugPrint(w io.Writer) {
	fmt.Fprintf(w, "%v %v %d\n", f.Perms, f.Name, len(f.Contents))
}

func (f *File) setAttribute(name string, args []*parse.Arg) error {
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

func (f *File) setPerms(args []*parse.Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	f.Perms = mode
	return nil
}

func (f *File) setTemplate(args []*parse.Arg) error {
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

func (f *File) setContents(args []*parse.Arg) error {
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
