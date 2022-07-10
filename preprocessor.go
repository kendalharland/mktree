package mktree

import (
	"io"
	"io/ioutil"
	"strings"
)

func preprocess(r io.Reader, substitutions map[string]string) (io.Reader, error) {
	input, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	output := string(input)
	for o, n := range substitutions {
		o = "%(" + o + ")"
		output = strings.ReplaceAll(output, o, n)
	}

	return strings.NewReader(output), nil
}
