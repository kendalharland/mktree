package mktree

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sys/unix"
)

func Example_dir() {
	interpret(`(dir "test")`)
	// Output:
	// drwxrwxrwx [example]
	// drwxrwxrwx [example]/test
}

func Example_file() {
	interpret(`(file "test")`)
	// Output:
	// drwxrwxrwx [example]
	// -rw-rw-rw- [example]/test 0
}

func Example_perms() {
	interpret(`
	(dir "a" (@perms 0700))
	(file "b" (@perms 0755))
	`)
	// Output:
	// drwxrwxrwx [example]
	// drwx------ [example]/a
	// -rwxr-xr-x [example]/b 0
}

func Example_tree_with_perms() {
	interpret(`
	(dir "a" 
		(@perms 0700)
		(file "b"
			(@perms 0755)))
	`)
	// Output:
	// drwxrwxrwx [example]
	// drwx------ [example]/a
	// -rwxr-xr-x [example]/a/b 0
}

func Example_vars() {
	interpretVars(`
	(dir "%(dirname)" (@perms %(perms)))
	(file "%(filename)" (@perms %(perms)))
	(dir "%(root_dir)")
	`, map[string]string{
		"dirname":  "a",
		"filename": "b.txt",
		"perms":    "0555",
	})
	// Output:
	// drwxrwxrwx [example]
	// dr-xr-xr-x [example]/a
	// drwxrwxrwx [example]/[example]
	// -r-xr-xr-x [example]/b.txt 0
}

func Example_parent_dirs() {
	interpret(`(file "/a/b/c/d/e.txt")`)
	// Output:
	// drwxrwxrwx [example]
	// -rw-rw-rw- [example]/a/b/c/d/e.txt 0
}

func interpret(source string) {
	interpretVars(source, nil)
}

func interpretVars(source string, vars map[string]string) {
	i := &Interpreter{
		Vars: vars,
		Root: "[example]",
	}
	dir, err := i.Interpret(strings.NewReader(source))
	if err != nil {
		panic(err)
	}
	dir.DebugPrint(os.Stdout)
}

// Executes examples/examples.tree and asserts the output is as expected.
func TestExamples(t *testing.T) {
	root, err := ioutil.TempDir("", "mktree")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)

	i := &Interpreter{
		Root: root,
		Vars: map[string]string{"my_var": "test"},
	}
	opts := []Option{
		WithTemplateFunction("FileExists", func(_ string) bool { return false }),
		WithTemplateFunction("FileContents", func(_ string) string { return "contents" }),
		WithTemplateFunction("Now", func() string { return "2022-03-01" }),
		WithTemplateFunction("User", func() string { return "test" }),
	}

	if err := i.ExecFile("examples/examples.tree", opts...); err != nil {
		t.Fatal(err)
	}

	assertDir(t, filepath.Join(root, "example"), defaultDirMode)
	assertFile(t, filepath.Join(root, "example.txt"), os.FileMode(0667), "")
	assertFile(t, filepath.Join(root, "template_example.txt"), defaultFileMode, strings.TrimSpace(`
[start:now_example]
The current time is 2022-03-01
[end:now_example]

[start:user_example]
The current user is test
[end:user_example]

[start:file_contents_example]
The file "VERSION" contains "contents"
[end:file_contents_example]

[start:file_exists_example]
The file "missing.txt" does not exist
[end:file_exists_example]

[start:var_example]
%(my_var) = test
[end:var_example]
`))
}

func assertDir(t *testing.T, name string, mode os.FileMode) {
	t.Helper()
	stat, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if !stat.IsDir() {
		t.Fatalf("not a directory: %s", name)
	}
	umask := unix.Umask(0)
	defer unix.Umask(umask)
	// mode = os.FileMode(uint32(umask) ^ uint32(mode))
	// fmt.Printf("umask=%#o, mode=%#o\n", umask, mode)
	if mode != stat.Mode() {
		t.Fatalf("expected dir mode %#o but got %#o", mode, stat.Mode())
	}
}

func assertFile(t *testing.T, name string, mode os.FileMode, contents string) {
	t.Helper()
	stat, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if stat.IsDir() {
		t.Fatalf("%s is a directory", name)
	}
	if stat.Mode() != mode {
		t.Fatalf("expected file mode %#o but got %#o", mode, stat.Mode())
	}
	got, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(got), contents); diff != "" {
		t.Fatalf("got contents diff (+got,-want):\n%s\n", diff)
	}
}
