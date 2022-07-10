package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/kendalharland/mktree"
)

func main() {
	if err := execute(context.TODO()); err != nil {
		log.Fatal(err)
	}
}

func usage() {

}

func execute(ctx context.Context) error {
	var root string
	args := RepeatedFlag(func() flag.Value { return &keyValueFlag{} })

	flag.StringVar(&root, "root", "", "Where to create the files and directories (defaults to cwd")
	flag.Var(args, "vars", "A list of key-value pairs to substitute in the source while preprocessing")
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
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

	mktree.DebugDir(d, os.Stdout)
	return nil
}
