package mktree

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestExecTemplateFile(t *testing.T) {
	f, err := ioutil.TempFile("", "mktree.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.RemoveAll(f.Name())

	template := "Hello {{ CustomFunction }}"
	if err := ioutil.WriteFile(f.Name(), []byte(template), os.FileMode(0777)); err != nil {
		t.Fatal(err)
	}

	funcMap := map[string]interface{}{
		"CustomFunction": func() string { return "Tester" },
	}
	content, err := execTemplateFile(f.Name(), funcMap)
	if err != nil {
		t.Fatal(err)
	}
	if content != "Hello Tester" {
		t.Fatalf("wanted 'Hello Tester' but got %q", content)
	}
}
