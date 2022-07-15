package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func execute() error {
	flag.Parse()
	if flag.NArg() != 1 {
		return errors.New("expected the path to docs")
	}

	docsPath := flag.Arg(0)
	if err := copyCLIUsageToDocs(docsPath); err != nil {
		return err
	}
	if err := copyReleaseNotesToDocs(docsPath); err != nil {
		return err
	}
	return nil
}

func copyCLIUsageToDocs(docsPath string) error {
	cmd := exec.Command("./bin/mktree")
	helptext, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("running %v: %w", cmd.Args, err)
	}

	filename := filepath.Join(docsPath, "posts/reference/index.html")
	doc, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("opening %v: %w", filename, err)
	}

	usage := html.EscapeString(string(helptext))
	output := strings.Replace(string(doc), "%(snippet cli-usage)", usage, -1)
	if err := ioutil.WriteFile(filename, []byte(output), 0666); err != nil {
		return fmt.Errorf("writing %v: %w", filename, err)
	}
	return nil
}

func copyReleaseNotesToDocs(docsPath string) error {
	cl, err := ioutil.ReadFile("CHANGELOG")
	if err != nil {
		return err
	}

	terminator := []byte("# [Releases]")
	changelog := make([]byte, len(cl))
	for i := len(cl) - 1; i >= 0; i-- {
		changelog[i] = cl[i]
		if bytes.HasPrefix(changelog[i:], terminator) {
			changelog = changelog[i+len(terminator):]
			break
		}
	}

	filename := filepath.Join(docsPath, "posts/changelog/index.html")
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading %v: %w", filename, err)
	}

	extensions := parser.CommonExtensions
	parser := parser.NewWithExtensions(extensions)
	html := markdown.ToHTML(changelog, parser, nil)

	output := strings.Replace(string(input), "%(snippet release-notes)", string(html), -1)
	if err := ioutil.WriteFile(filename, []byte(output), 0666); err != nil {
		return fmt.Errorf("writing %v: %w", filename, err)
	}
	return nil
}