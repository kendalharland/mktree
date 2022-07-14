package mktree

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInterpreter_Interpret(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		source  string
		vars    map[string]string
		want    []interface{}
		wantErr error
	}{
		// OK cases.
		{
			name: "comments",
			source: `
            ; comment line 1
            (dir "a")
            
            ; comment line 2
            (file "b")
			`,
			want: []interface{}{
				&Dir{Name: "a", Perms: defaultDirMode},
				&File{Name: "b", Perms: defaultFileMode},
			},
		},
		// Dir
		{
			name:   "dir",
			source: `(dir "a")`,
			want: []interface{}{
				&Dir{Name: "a", Perms: defaultDirMode},
			},
		},
		{
			name:   "dir_and_file",
			source: `(dir "a") (file "b")`,
			want: []interface{}{
				&Dir{Name: "a", Perms: defaultDirMode},
				&File{Name: "b", Perms: defaultFileMode},
			},
		},
		{
			name:   "dir_with_perms",
			source: `(dir "a" (@perms 0555))`,
			want: []interface{}{
				&Dir{Name: "a", Perms: os.FileMode(0555) | os.ModeDir},
			},
		},
		{
			name:   "dir_with_file",
			source: `(dir "a" (@perms 0555))`,
			want: []interface{}{
				&Dir{Name: "a", Perms: os.FileMode(0555) | os.ModeDir},
			},
		},
		// File
		{
			name:   "file",
			source: `(file "a")`,
			want: []interface{}{
				&File{Name: "a", Perms: defaultFileMode},
			},
		},
		{
			name:   "file_with_perms",
			source: `(file "a" (@perms 0712))`,
			want: []interface{}{
				&File{Name: "a", Perms: os.FileMode(0712)},
			},
		},
		{
			name:   "file_with_contents",
			source: `(file "a" (@contents "this is a test"))`,
			want: []interface{}{
				&File{Name: "a", Contents: []byte("this is a test"), Perms: defaultFileMode},
			},
		},
		{
			name:   "file_with_template",
			source: `(file "a" (@template "template.tmpl"))`,
			want: []interface{}{
				&File{Name: "a", TemplateFilename: "template.tmpl", Perms: defaultFileMode},
			},
		},
		{
			name: "paths_are_relative_to_parent",
			root: "/root",
			source: `(file "/a")
			         (dir  "/b" (file "/c"))`,
			want: []interface{}{
				&File{Name: "/root/a", Perms: defaultFileMode},
				&Dir{Name: "/root/b", Perms: defaultDirMode, Files: []*File{
					{Name: "/root/b/c", Perms: defaultFileMode},
				}},
			},
		},
		{
			name: "paths_are_cleaned",
			source: `(file "///////a/b///c/////d")
			         (dir "///d/e///f///")`,
			want: []interface{}{
				&File{Name: "/a/b/c/d", Perms: defaultFileMode},
				&Dir{Name: "/d/e/f", Perms: defaultDirMode},
			},
		},
		// Error cases.
		{
			name:    "file_missing_name",
			source:  `(file)`,
			wantErr: ErrInterpret,
		},
		{
			name:    "dir_missing_name",
			source:  `(dir)`,
			wantErr: ErrInterpret,
		},
		{
			name:    "dir_perms_missing_name",
			source:  `(dir (@perms 0555))`,
			wantErr: ErrInterpret,
		},
		{
			name:    "file_perms_missing_name",
			source:  `(file (@perms 0712))`,
			wantErr: ErrInterpret,
		},

		{
			name:    "dir_perms_invalid_file_mode_type",
			source:  `(dir "foo" (@perms "nan"))`,
			wantErr: ErrInterpret,
		},
		{
			name:    "file_perms_invalid_file_mode_type",
			source:  `(file (@perms "nan"))`,
			wantErr: ErrInterpret,
		},
		{
			name:    "file_template_is_mutually_exclusive_with_contents",
			source:  `(file "a" (@content "this is a") (@template "a.tmpl"))`,
			wantErr: ErrInterpret,
		},
		{
			name:    "file_contents_is_mutually_exclusive_with_template",
			source:  `(file "a" (@template "a.tmpl") (@content "this is a"))`,
			wantErr: ErrInterpret,
		},
		{
			name:    "dir_perms_invalid_neg_file_mode",
			source:  `(dir "a" (@perms -1))`, // Grammar excludes negative ints.
			wantErr: ErrSyntax,
		},
		{
			name:    "file_perms_invalid_neg_file_mode",
			source:  `(file "a" (@perms -1))`,
			wantErr: ErrSyntax,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			i := &Interpreter{
				Vars:   test.vars,
				Root:   test.root,
				Stderr: &stderr,
			}

			want, err := mkdir(test.root, test.want)
			if err != nil {
				t.Fatal(err)
			}

			d, err := i.Interpret(strings.NewReader(test.source))
			t.Log(stderr.String())
			switch {
			case err != nil && test.wantErr != nil:
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Interpret(`%s`) wanted a %v but got %v", test.source, test.wantErr, err)
				}
				return
			case err != nil && test.wantErr == nil:
				t.Fatalf("Interpret(`%s`) got unexpected error: %v", test.source, err)
			case err == nil && test.wantErr != nil:
				t.Fatalf("Interpret(`%s`) wanted error but got %+v", test.source, d)
			}

			if diff := cmp.Diff(want, d); diff != "" {
				t.Fatalf("Interpret(`%s`) got diff (+got,-want):\n%s\n", test.source, diff)
			}
		})
	}
}

func mkdir(root string, entries []interface{}) (*Dir, error) {
	d := defaultRootDir(root)
	for i, e := range entries {
		switch t := e.(type) {
		case *File:
			d.Files = append(d.Files, t)
		case *Dir:
			d.Dirs = append(d.Dirs, t)
		default:
			return nil, fmt.Errorf("value %v of type %#T at position %d is not a *File or a *Dir", e, e, i)
		}
	}
	return d, nil
}
