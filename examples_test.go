package mktree

import (
	"os"
	"strings"
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
		"perms":    "555",
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
		Root: "[example]",
		Vars: vars,
	}
	dir, err := i.Interpret(strings.NewReader(source))
	if err != nil {
		panic(err)
	}
	dir.DebugPrint(os.Stdout)
}
