package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	if err := copyCodeSnippetsToDocs(docsPath); err != nil {
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
	output := strings.Replace(string(doc), "%(content cli-usage)", usage, -1)
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

	// Copy everything after "# [Releases]"
	terminator := []byte("# [Releases]")
	changelog := make([]byte, len(cl))
	for i := len(cl) - 1; i >= 0; i-- {
		changelog[i] = cl[i]
		if bytes.HasPrefix(changelog[i:], terminator) {
			changelog = changelog[i+len(terminator):]
			break
		}
	}

	m := regexp.MustCompile(`\[(\d.\d.\d)\]`)
	tagsURLPrefix := "https://github.com/kendalharland/mktree/releases/tag/v"
	replacement := fmt.Sprintf("[${1}](%s${1})", tagsURLPrefix)
	changelog = []byte(m.ReplaceAllString(string(changelog), replacement))

	extensions := parser.CommonExtensions
	parser := parser.NewWithExtensions(extensions)
	html := string(markdown.ToHTML(changelog, parser, nil))

	filename := filepath.Join(docsPath, "posts/changelog/index.html")
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading %v: %w", filename, err)
	}

	// Replace the meta heading.
	output := strings.ReplaceAll(string(input), `content="%(content release-notes)"`, `content="release-notes"`)
	// Replace the actual heading.
	output = strings.ReplaceAll(output, `%(content release-notes)`, html)
	if err := ioutil.WriteFile(filename, []byte(output), 0666); err != nil {
		return fmt.Errorf("writing %v: %w", filename, err)
	}
	return nil
}

func copyCodeSnippetsToDocs(docsPath string) error {
	root := filepath.Join(docsPath, "posts")
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		contents, err := inlineCodeSnippetsInFile(path)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(path, contents, d.Type())
	})
}

func inlineCodeSnippetsInFile(filename string) ([]byte, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`%\(snippet (.*)\)`)
	matches := re.FindAllStringSubmatch(string(contents), -1)
	for _, m := range matches {
		options := strings.Fields(m[1])
		tag := options[0]
		filename := options[1]
		replacement, err := extractTagRegionFromFile(tag, filename)
		if err != nil {
			return nil, err
		}
		original := []byte(m[0])
		contents = bytes.ReplaceAll(contents, original, replacement)
	}

	return contents, nil
}

func extractTagRegionFromFile(region, filename string) ([]byte, error) {
	var lines []string
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	startLine := fmt.Sprintf("// start:%s", region)
	endLine := fmt.Sprintf("// end:%s", region)
	var inRegion bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == startLine {
			inRegion = true
			continue
		}
		if line == endLine {
			break
		}
		if inRegion {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	contents := []byte(strings.Join(lines, "\n"))
	return contents, nil
}
