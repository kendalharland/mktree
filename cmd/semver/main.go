package main

// TODO: Check patch, minor, major

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/maruel/subcommands"
)

func main() {
	a := &subcommands.DefaultApplication{
		Name:  "release-tool",
		Title: "A tool for managing mktree releases",
		Commands: []*subcommands.Command{
			cmdBumpVersionBuild,
			subcommands.CmdHelp,
		},
	}

	os.Exit(subcommands.Run(a, nil))
}

var cmdBumpVersionBuild = &subcommands.Command{
	UsageLine: "bump-version <version-file>",
	ShortDesc: "Bump the build number",
	LongDesc:  "Bump the build number",
	CommandRun: func() subcommands.CommandRun {
		c := &runBumpVersion{}
		c.Flags.BoolVar(&c.patch, "patch", false, "Increment the patch number")
		c.Flags.BoolVar(&c.minor, "minor", false, "Increment the minor number")
		c.Flags.BoolVar(&c.major, "major", false, "Increment the major number")
		return c
	},
}

type runBumpVersion struct {
	subcommands.CommandRunBase

	patch, minor, major bool
}

func (c *runBumpVersion) Run(a subcommands.Application, args []string, env subcommands.Env) int {
	if err := c.run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func (c *runBumpVersion) run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected one argument")
	}
	if !xor(c.patch, c.minor, c.major) {
		return fmt.Errorf("expected just one of -patch -minor or -major")
	}

	f := args[0]
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}
	version := semver.MustParse(strings.TrimSpace(string(b)))

	var next semver.Version
	switch {
	case c.patch:
		next = version.IncPatch()
	case c.minor:
		next = version.IncMinor()
	case c.major:
		next = version.IncMajor()
	}

	return ioutil.WriteFile(f, []byte(next.String()), os.FileMode(0666))
}

func xor(values ...bool) bool {
	var foundTrue bool
	for _, value := range values {
		if value && foundTrue {
			return false
		}
		if value {
			foundTrue = true
		}
	}
	return foundTrue
}
