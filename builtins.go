package mktree

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"time"
)

var builtin = []Option{
	WithTemplateFunction("FileExists", newFileExistsBuiltin),
	WithTemplateFunction("FileContents", newFileContentsBuiltin),
	WithTemplateFunction("Now", newNowBuiltin()),
	WithTemplateFunction("User", newUserBuiltin()),
}

func newFileContentsBuiltin(filename string) func() (string, error) {
	return func() (string, error) {
		contents, err := ioutil.ReadFile(filename)
		return string(contents), err
	}
}

func newFileExistsBuiltin(filename string) func() bool {
	return func() bool {
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

func warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "warning: "+format, args...)
}
