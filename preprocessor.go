package mktree

import (
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

	vs, err := getVars(input)
	if err != nil {
		return nil, err
	}
	if !allowUndefined {
		for _, v := range vs {
			if _, ok := vars[v]; !ok {
				return nil, fmt.Errorf("undefined variable: %s", v)
			}
		}
	}

	output := string(input)
	for o, n := range vars {
		o = "%(" + o + ")"
		output = strings.ReplaceAll(output, o, n)
	}

	return strings.NewReader(output), nil
}

func getVars(input []byte) ([]string, error) {
	var vars []string

	re := regexp.MustCompile(`%\([^)]*\)`)
	matches := re.FindAll(input, -1)
	for _, match := range matches {
		m := string(match)
		if m == "%()" {
			return nil, errors.New("empty variable pattern '%()' is not allowed")
		}
		vars = append(vars, m[2:len(m)-1])
	}
	return vars, nil
}
