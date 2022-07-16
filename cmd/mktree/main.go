package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kendalharland/mktree"

	_ "embed"
)

const helpext = `
usage: mktree [-debug] [-version] [-allow-undefined-vars]
              [-vars=<name>=<value>]
              <source-file>
`

func parseFlags() *options {
	flag.Usage = usage

	o := &options{vars: &variablesFlag{}}
	flag.BoolVar(&o.debug, "debug", false, "Print the results without creating any files or directories")
	flag.BoolVar(&o.version, "version", false, "Print the version and exit")
	flag.BoolVar(&o.allowUndefinedVars, "allow-undefined-vars", false, "Allow undefined variables in the input")
	flag.Var(o.vars, "vars", "A list of key-value pairs to substitute in the source while preprocessing")
	flag.Parse()
	return o
}

type options struct {
	debug              bool
	version            bool
	allowUndefinedVars bool
	vars               flag.Getter
}

func main() {
	if err := execute(context.TODO()); err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Println(strings.TrimSpace(helpext))
	fmt.Println()
	flag.PrintDefaults()
}

func printVersion() {
	e, _ := os.Executable()
	name := filepath.Base(e)
	fmt.Fprintf(os.Stdout, "%s %s\n", name, mktree.Version())
}

func execute(ctx context.Context) error {
	o := parseFlags()

	if o.version {
		printVersion()
		return nil
	}

	if flag.NArg() == 0 {
		flag.Usage()
		return nil
	}

	input, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		return err
	}

	i := &mktree.Interpreter{
		Vars:               o.vars.Get().(map[string]string),
		AllowUndefinedVars: o.allowUndefinedVars,
	}

	tree, err := i.Interpret(bytes.NewReader(input))
	if err != nil {
		return err
	}

	if o.debug {
		tree.DebugPrint(os.Stdout)
		return nil
	}

	return tree.Create()
}
