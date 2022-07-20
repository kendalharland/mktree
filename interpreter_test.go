package mktree

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kendalharland/mktree/parse"
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
				&dir{name: "[test_root]/a", perms: defaultDirMode},
				&file{name: "[test_root]/b", perms: defaultFileMode},
			},
		},
		// Dir
		{
			name:   "dir",
			source: `(dir "a")`,
			want: []interface{}{
				&dir{name: "[test_root]/a", perms: defaultDirMode},
			},
		},
		{
			name:   "dir_and_file",
			source: `(dir "a") (file "b")`,
			want: []interface{}{
				&dir{name: "[test_root]/a", perms: defaultDirMode},
				&file{name: "[test_root]/b", perms: defaultFileMode},
			},
		},
		{
			name:   "dir_with_perms",
			source: `(dir "a" (@perms 0555))`,
			want: []interface{}{
				&dir{name: "[test_root]/a", perms: os.FileMode(0555) | os.ModeDir},
			},
		},
		{
			name:   "dir_with_file",
			source: `(dir "a" (@perms 0555))`,
			want: []interface{}{
				&dir{name: "[test_root]/a", perms: os.FileMode(0555) | os.ModeDir},
			},
		},
		// File
		{
			name:   "file",
			source: `(file "a")`,
			want: []interface{}{
				&file{name: "[test_root]/a", perms: defaultFileMode},
			},
		},
		{
			name:   "file_with_perms",
			source: `(file "a" (@perms 0712))`,
			want: []interface{}{
				&file{name: "[test_root]/a", perms: os.FileMode(0712)},
			},
		},
		{
			name:   "file_with_contents",
			source: `(file "a" (@contents "this is a test"))`,
			want: []interface{}{
				&file{name: "[test_root]/a", contents: []byte("this is a test"), perms: defaultFileMode},
			},
		},
		{
			name:   "file_with_template",
			source: `(file "a" (@template "template.tmpl"))`,
			want: []interface{}{
				&file{name: "[test_root]/a", TemplateFilename: "template.tmpl", perms: defaultFileMode},
			},
		},
		{
			name: "paths_are_relative_to_parent",
			root: "/root",
			source: `(file "/a")
			         (dir  "/b" (file "/c"))`,
			want: []interface{}{
				&file{name: "/root/a", perms: defaultFileMode},
				&dir{name: "/root/b", perms: defaultDirMode, files: []*file{
					{name: "/root/b/c", perms: defaultFileMode},
				}},
			},
		},
		{
			name: "paths_are_cleaned",
			source: `(file "///////a/b///c/////d")
			         (dir "///d/e///f///")`,
			want: []interface{}{
				&file{name: "[test_root]/a/b/c/d", perms: defaultFileMode},
				&dir{name: "[test_root]/d/e/f", perms: defaultDirMode},
			},
		},
		{
			name:   "root_dir_var_is_set_to_root_by_default",
			source: `(file "%(root_dir)/a")`,
			want: []interface{}{
				&file{name: "[test_root]/[test_root]/a", perms: defaultFileMode},
			},
		},
		{
			name:   "root_dir_var_is_set_to_user_value",
			vars:   map[string]string{"root_dir": "test"},
			source: `(file "%(root_dir)/a")`,
			want: []interface{}{
				&file{name: "[test_root]/test/a", perms: defaultFileMode},
			},
		},
		{
			name: "var_whitespace_padding_is_stripped",
			vars: map[string]string{"my_var": "value"},
			source: `
			(file "a" (@contents "%(  my_var     )"))
			(file "b" (@contents "%(my_var)"))
			`,
			want: []interface{}{
				&file{name: "[test_root]/a", contents: []byte("value"), perms: defaultFileMode},
				&file{name: "[test_root]/b", contents: []byte("value"), perms: defaultFileMode},
			},
		},
		// Link
		{
			name: "link",
			source: `
			(file "target")
			(link "target" "the_link")
			`,
			want: []interface{}{
				&file{name: "[test_root]/target", perms: defaultFileMode},
				&link{name: "[test_root]/the_link", target: "[test_root]/target"},
			},
		},
		{
			name: "symlink",
			source: `
			(file "target")
			(link "target" "the_link" (@symbolic))
			`,
			want: []interface{}{
				&file{name: "[test_root]/target", perms: defaultFileMode},
				&link{name: "[test_root]/the_link", target: "[test_root]/target", symbolic: true},
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
			name:    "file_perms_invalid_file_mode_number",
			source:  `(file (@perms 555))`,
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
			wantErr: parse.ErrSyntax,
		},
		{
			name:    "file_perms_invalid_neg_file_mode",
			source:  `(file "a" (@perms -1))`,
			wantErr: parse.ErrSyntax,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := test.root
			if root == "" {
				root = "[test_root]"
			}

			var stderr bytes.Buffer
			i := &Interpreter{
				Root:   root,
				Vars:   test.vars,
				Stderr: &stderr,
			}

			want, err := mkdir(root, test.want)
			if err != nil {
				t.Fatal(err)
			}

			tree, err := i.Interpret(strings.NewReader(test.source))
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
				t.Fatalf("Interpret(`%s`) wanted error but got %+v", test.source, tree.root)
			}

			if diff := cmp.Diff(want, tree.root, cmp.AllowUnexported(dir{}, file{}, link{})); diff != "" {
				t.Fatalf("Interpret(`%s`) got diff (+got,-want):\n%s\n", test.source, diff)
			}
		})
	}
}

func mkdir(root string, entries []interface{}) (*dir, error) {
	d := defaultRootDir(root)
	// TODO: Dedup with the same setter in interpreter.go.
	for i, e := range entries {
		switch t := e.(type) {
		case *file:
			d.addFile(t)
		case *dir:
			d.addDir(t)
		case *link:
			d.addLink(t)
		default:
			return nil, fmt.Errorf("value %v of type %#T at position %d is not a *File or a *Dir", e, e, i)
		}
	}
	return d, nil
}
