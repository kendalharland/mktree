package mktree

import (
	"io/fs"
	"os"
)

type Dir struct {
	Name  string
	Perms os.FileMode
	Files []*File
	Dirs  []*Dir
}

func (d *Dir) addDir(child *Dir) {
	d.Dirs = append(d.Dirs, child)
}

func (d *Dir) addFile(child *File) {
	d.Files = append(d.Files, child)
}

func (d *Dir) setAttribute(name string, args []*Arg) error {
	switch name {
	case "perms":
		return d.setPerms(args)
	}
	return interpretError("invalid file attribute %q", name)
}

func (d *Dir) setPerms(args []*Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	d.Perms = mode | fs.ModeDir
	return nil
}
