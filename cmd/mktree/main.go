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

	o := &options{}
	o.vars = &repeatedFlag{value: func() flag.Value { return &keyValueFlag{} }}
	flag.BoolVar(&o.debug, "debug", false, "Print the results without creating any files or directories")
	flag.BoolVar(&o.version, "version", false, "Print the version and exit")
	flag.BoolVar(&o.allowUndefinedVars, "allow-undefined-vars", false, "Allow undefined variables in the input")
	flag.Var(o.vars, "vars", "A list of key-value pairs to substitute in the source while preprocessing")
	flag.Parse()
	return o
}

type options struct {
	root               string
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

	subs := map[string]string{}
	kvs := o.vars.Get().([]flag.Value)
	for _, v := range kvs {
		kv := v.(*keyValueFlag)
		subs[kv.K] = kv.V
	}

	i := &mktree.Interpreter{
		Vars:               subs,
		Root:               o.root,
		AllowUndefinedVars: o.allowUndefinedVars,
	}

	d, err := i.Interpret(bytes.NewReader(input))
	if err != nil {
		return err
	}

	if o.debug {
		mktree.DebugDir(d, os.Stdout)
		return nil
	}

	return mktree.GenerateDir(d)
}
