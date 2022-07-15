package mktree

import _ "embed"

//go:embed VERSION
var version []byte

// Version returns the current version of this package.
func Version() string { return string(version) }
