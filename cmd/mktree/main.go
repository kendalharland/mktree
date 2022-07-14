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

	"github.com/kendalharland/mktree"

	_ "embed"
)

func main() {
	if err := execute(context.TODO()); err != nil {
		log.Fatal(err)
	}
}

func usage() {

}

func printVersion() {
	e, _ := os.Executable()
	name := filepath.Base(e)
	fmt.Fprintf(os.Stdout, "%s %s\n", name, mktree.Version())
}

func execute(ctx context.Context) error {
	var (
		root               string
		debug              bool
		version            bool
		allowUndefinedVars bool
	)
	args := &repeatedFlag{value: func() flag.Value { return &keyValueFlag{} }}

	flag.BoolVar(&debug, "debug", false, "Print the results without creating any files or directories")
	flag.BoolVar(&version, "version", false, "Print the version and exit")
	flag.BoolVar(&allowUndefinedVars, "allow-undefined-vars", false, "Allow undefined variables in the input")
	flag.Var(args, "vars", "A list of key-value pairs to substitute in the source while preprocessing")
	flag.Parse()

	if version {
		printVersion()
		return nil
	}

	if flag.NArg() == 0 {
		usage()
		return nil
	}

	input, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		return err
	}

	subs := map[string]string{}
	kvs := args.Get().([]flag.Value)
	for _, v := range kvs {
		kv := v.(*keyValueFlag)
		subs[kv.K] = kv.V
	}

	i := &mktree.Interpreter{
		Vars:               subs,
		Root:               root,
		AllowUndefinedVars: allowUndefinedVars,
	}

	d, err := i.Interpret(bytes.NewReader(input))
	if err != nil {
		return err
	}

	if debug {
		mktree.DebugDir(d, os.Stdout)
		return nil
	}

	return mktree.GenerateDir(d)
}
