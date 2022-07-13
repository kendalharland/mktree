package mktree

import (
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

		// Error cases.
		{
			name:    "file_missing_name",
			source:  `(file)`,
			wantErr: InterpretError{},
		},
		{
			name:    "dir_missing_name",
			source:  `(dir)`,
			wantErr: InterpretError{},
		},
		{
			name:    "dir_perms_missing_name",
			source:  `(dir (@perms 0555))`,
			wantErr: InterpretError{},
		},
		{
			name:    "file_perms_missing_name",
			source:  `(file (@perms 0712))`,
			wantErr: InterpretError{},
		},

		{
			name:    "dir_perms_invalid_file_mode_type",
			source:  `(dir "foo" (@perms "nan"))`,
			wantErr: InterpretError{},
		},
		{
			name:    "file_perms_invalid_file_mode_type",
			source:  `(file (@perms "nan"))`,
			wantErr: InterpretError{},
		},
		{
			name:    "dir_perms_invalid_neg_file_mode",
			source:  `(dir "a" (@perms -1))`, // Grammar excludes negative ints.
			wantErr: ParseError{},
		},
		{
			name:    "file_perms_invalid_neg_file_mode",
			source:  `(file "a" (@perms -1))`,
			wantErr: ParseError{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			i := &Interpreter{
				Vars: test.vars,
			}

			want, err := mkdir(test.want)
			if err != nil {
				t.Fatal(err)
			}

			d, err := i.Interpret(strings.NewReader(test.source))
			switch {
			case err != nil && test.wantErr != nil:
				if !errors.As(err, &test.wantErr) {
					t.Fatalf("Interpret(`%s`) wanted a %#T but got %v", test.source, test.wantErr, err)
				}
				return
			case err != nil && test.wantErr == nil:
				t.Fatalf("Interpret(`%s`) got unexpected error: %v", test.source, err)
			case err == nil && test.wantErr != nil:
				t.Fatalf("Interpret(`%s`) wanted error but got %+v", test.source, d)
			}

			if diff := cmp.Diff(d, want); diff != "" {
				t.Fatalf("Interpret(`%s`) got diff (+got,-want):\n%s\n", test.source, diff)
			}
		})
	}
}

func mkdir(entries []interface{}) (*Dir, error) {
	root := defaultRootDir("")
	for i, e := range entries {
		switch t := e.(type) {
		case *File:
			root.Files = append(root.Files, t)
		case *Dir:
			root.Dirs = append(root.Dirs, t)
		default:
			return nil, fmt.Errorf("value %v of type %#T at position %d is not a *File or a *Dir", e, e, i)
		}
	}
	return root, nil
}
