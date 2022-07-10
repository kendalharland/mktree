package mktree

import (
	"fmt"
	"io"
)

func DebugDir(root *Dir, w io.Writer) {
	fmt.Fprintf(w, "%v %v\n", root.Perms, root.Name)
	for _, d := range root.Dirs {
		DebugDir(d, w)
	}
	for _, f := range root.Files {
		DebugFile(f, w)
	}
}

func DebugFile(f *File, w io.Writer) {
	fmt.Fprintf(w, "%v %v %d\n", f.Perms, f.Name, len(f.Contents))
}
