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
	name  string
	perms os.FileMode
	files []*file
	dirs  []*dir
	links []*link
}

func (d *dir) debugPrint(w io.Writer) {
	fmt.Fprintf(w, "%v %v\n", d.perms, d.name)
	for _, child := range d.dirs {
		child.debugPrint(w)
	}
	for _, child := range d.files {
		child.debugPrint(w)
	}
}

func (d *dir) addDir(child *dir) {
	d.dirs = append(d.dirs, child)
}

func (d *dir) addFile(child *file) {
	d.files = append(d.files, child)
}

func (d *dir) addLink(child *link) {
	d.links = append(d.links, child)
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
	d.perms = mode | fs.ModeDir
	return nil
}

type file struct {
	name             string
	perms            os.FileMode
	contents         []byte
	TemplateFilename string
}

func (f *file) debugPrint(w io.Writer) {
	fmt.Fprintf(w, "%v %v %d\n", f.perms, f.name, len(f.contents))
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
	f.perms = mode
	return nil
}

func (f *file) setTemplate(args []*parse.Arg) error {
	if len(f.contents) > 0 {
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
	f.contents = []byte(contents)
	return nil
}

type link struct {
	name     string
	target   string
	symbolic bool
}

func (l *link) setAttribute(name string, args []*parse.Arg) error {
	switch name {
	case "symbolic":
		l.symbolic = true
	}
	return nil
}
