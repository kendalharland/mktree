package mktree

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"time"
)

func builtins(i *Interpreter) []Option {
	return []Option{
		WithTemplateFunction("FileExists", newFileExistsBuiltin()),
		WithTemplateFunction("FileContents", newFileContentsBuiltin()),
		WithTemplateFunction("Now", newNowBuiltin()),
		WithTemplateFunction("User", newUserBuiltin()),
		WithTemplateFunction("Var", newVarBuiltin(i.Vars)),
	}
}

func newFileContentsBuiltin() func(string) (string, error) {
	return func(filename string) (string, error) {
		contents, err := ioutil.ReadFile(filename)
		return string(contents), err
	}
}

func newFileExistsBuiltin() func(string) bool {
	return func(filename string) bool {
		stat, err := os.Stat(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				warn("unable to stat %s: %w", filename, err)
			}
			return false
		}
		return !stat.IsDir()
	}
}

func newNowBuiltin() func() string {
	t := time.Now()
	now := t.Format(time.RFC3339)
	return func() string { return now }
}

func newUserBuiltin() func() string {
	var u string
	if usr, err := user.Current(); err != nil {
		warn("unable to get current user")
	} else {
		u = usr.Username
		if u == "" {
			u = usr.Name
		}
	}
	return func() string { return u }
}

func newVarBuiltin(vars map[string]string) func(string) (string, error) {
	return func(varname string) (string, error) {
		if v, ok := vars[varname]; ok {
			return v, nil
		}
		return "", fmt.Errorf("variable %q is undefined", varname)
	}
}

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "warning: "+format, args...)
}
