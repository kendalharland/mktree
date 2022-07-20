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

func (d *dir) addChild(child interface{}) error {
	switch t := child.(type) {
	case *file:
		d.addFile(t)
	case *dir:
		d.addDir(t)
	case *link:
		d.addLink(t)
	default:
		return fmt.Errorf("%v is not a valid directory child", child)
	}
	return nil
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

func (d *dir) setPerms(args []*parse.Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	d.perms = mode | fs.ModeDir
	return nil
}

type file struct {
	name         string
	perms        os.FileMode
	contents     []byte
	templatePath string
}

func (f *file) debugPrint(w io.Writer) {
	fmt.Fprintf(w, "%v %v %d\n", f.perms, f.name, len(f.contents))
}

func (f *file) setPerms(perms os.FileMode) error {
	f.perms = perms
	return nil
}

func (f *file) setTemplate(filename string) error {
	if len(f.contents) > 0 {
		return errors.New("cannot set @template if @contents is set")
	}
	f.templatePath = filename
	return nil
}

func (f *file) setContents(contents []byte) error {
	if f.templatePath != "" {
		return errors.New("cannot set @contents if @template is set")
	}
	f.contents = []byte(contents)
	return nil
}

type link struct {
	name     string
	target   string
	symbolic bool
}
