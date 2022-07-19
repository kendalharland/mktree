package mktree

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"golang.org/x/sys/unix"
)

type Tree struct {
	root *Dir
}

func (t *Tree) DebugPrint(w io.Writer) {
	t.root.DebugPrint(w)
}

func (t *Tree) createDir(ctx *thread, d *Dir) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	if err := os.MkdirAll(d.Name, d.Perms); err != nil {
		return err
	}
	for _, child := range d.Dirs {
		if err := t.createDir(ctx, child); err != nil {
			return err
		}
	}
	for _, child := range d.Files {
		if err := t.createFile(ctx, child); err != nil {
			return err
		}
	}

	return nil
}

func (t *Tree) createFile(ctx *thread, f *File) error {
	umask := unix.Umask(0)
	defer unix.Umask(umask)

	contents, err := fileContents(ctx, f)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.Name), defaultDirMode); err != nil {
		return err
	}
	return ioutil.WriteFile(f.Name, []byte(contents), f.Perms)
}

func fileContents(ctx *thread, f *File) (string, error) {
	if len(f.Contents) > 0 {
		return string(f.Contents), nil
	}
	if len(f.TemplateFilename) > 0 {
		filename := filepath.Join(ctx.sourceRoot, f.TemplateFilename)
		return execTemplateFile(filename, ctx.templateFuncs)
	}
	return "", nil
}

func execTemplateFile(filename string, funcMap template.FuncMap) (string, error) {
	name := filepath.Base(filename)
	tmpl, err := template.New(name).Funcs(funcMap).ParseFiles(filename)
	if err != nil {
		return "", err
	}
	var contents bytes.Buffer
	if err := tmpl.Execute(&contents, nil); err != nil {
		return "", err
	}
	return contents.String(), nil
}
