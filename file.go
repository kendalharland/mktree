package mktree

import (
	"errors"
	"os"
)

type File struct {
	Name             string
	Perms            os.FileMode
	Contents         []byte
	TemplateFilename string
}

func (f *File) setAttribute(name string, args []*Arg) error {
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

func (f *File) setPerms(args []*Arg) error {
	mode, err := evalFileMode(args[0])
	if err != nil {
		return err
	}
	f.Perms = mode
	return nil
}

func (f *File) setTemplate(args []*Arg) error {
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

func (f *File) setContents(args []*Arg) error {
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
