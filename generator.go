package mktree

import (
	"os"
	"path/filepath"
)

func GenerateDir(d *Dir) error {
	if err := os.MkdirAll(d.Name, d.Perms); err != nil {
		return err
	}
	for _, child := range d.Dirs {
		if err := GenerateDir(child); err != nil {
			return err
		}
	}
	for _, child := range d.Files {
		if err := GenerateFile(child); err != nil {
			return err
		}
	}

	return nil
}

func GenerateFile(f *File) error {
	if err := os.MkdirAll(filepath.Dir(f.Name), defaultDirMode); err != nil {
		return err
	}
	fd, err := os.OpenFile(f.Name, os.O_CREATE|os.O_RDWR, f.Perms)
	if err != nil {
		return err
	}
	defer fd.Close()

	if len(f.Contents) > 0 {
		if _, err := fd.Write(f.Contents); err != nil {
			return err
		}
	}
	return nil
}
