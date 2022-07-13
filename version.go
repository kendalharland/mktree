package mktree

import _ "embed"

//go:embed VERSION
var version []byte

func Version() string {
	return string(version)
}
