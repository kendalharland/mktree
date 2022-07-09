package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
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

	flag.StringVar(&root, "root", "", "Where to create the files and directories (defaults to cwd")

	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}

	fd, err := os.Open(flag.Arg(0))
	if err != nil {
		return err
	}
	defer fd.Close()

	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("no root directory given and can't get the current working directory: %w", err)
		}
		root = cwd
	}

	return Execute(fd, root)
}
