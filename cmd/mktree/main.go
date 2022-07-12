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
)

const version = "0.0.0"

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
	fmt.Fprintf(os.Stdout, "%s %s\n", name, version)
}

func execute(ctx context.Context) error {
	mktree.Docs(os.Stdout)
	return nil

	var root string
	var debug bool
	var version bool
	args := &repeatedFlag{value: func() flag.Value { return &keyValueFlag{} }}

	flag.BoolVar(&debug, "debug", false, "Print the results without creating any files or directories")
	flag.StringVar(&root, "root", "", "Where to create the files and directories (defaults to cwd")
	flag.BoolVar(&version, "version", false, "Print the version and exit")

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

	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("no root directory given and can't get the current working directory: %w", err)
		}
		root = cwd
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
		Vars: subs,
		Root: root,
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
