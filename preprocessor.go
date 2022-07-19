package mktree

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
)

func preprocess(r io.Reader, vars map[string]string, allowUndefined bool) (io.Reader, error) {
	input, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	output := input

	re := regexp.MustCompile(`%\([^)]*\)`)
	matches := re.FindAll(input, -1)
	for _, match := range matches {
		m := string(match)
		if m == "%()" {
			return nil, errors.New("empty variable pattern '%()' is not allowed")
		}

		varname := m[2 : len(m)-1] // remove the surrounding %( and ).
		varname = strings.TrimSpace(varname)
		if value, ok := vars[varname]; !ok && !allowUndefined {
			// Use quotes incase v contains leading or trailing whitespace.
			return nil, fmt.Errorf("undefined variable: %q", varname)
		} else {
			output = bytes.ReplaceAll(output, []byte(m), []byte(value))
		}
	}

	return bytes.NewReader(output), nil
}
