#!/bin/bash

set -eu

DOCS_SRC_DIR=doc
DOCS_TMP_DIR=`pwd`/.docs-staging
DOCS_OUT_DIR=docs

mkdir -p $DOCS_TMP_DIR

function build {
  workdir=$(mktemp -d)

  echo "info: copying doc/ to $workdir"
  cp -r doc $workdir


  echo "info: running hugo"
  pushd $workdir/doc
  hugo
  popd

  echo "info: copying generated docs to $DOCS_OUT_DIR"
  rsync -av --progress --delete $workdir/doc/docs/ $DOCS_OUT_DIR
  rm -rf $workdir
}

function serve {
  build

  pushd $DOCS_SRC_DIR
  hugo server -D
  popd
}

while getopts "bps" COMMAND; do
  case $COMMAND in
  b) 
     build
     ;;
  s) 
     serve
     ;;
  *)
     echo "Invalid option"
     exit 1
     ;;
  esac
done

