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

func (d *Dir) setAttribute(name string, args []*Arg) error {
	value, err := evalNumber(args[0].Literal)
	if err != nil {
		return err
	}
	d.setPerms(os.FileMode(value))
	return nil
}

func (d *Dir) setPerms(m os.FileMode) {
	d.Perms = m | fs.ModeDir
}
