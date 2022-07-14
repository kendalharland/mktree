#!/bin/bash

set -eu

DOCS_SRC_DIR=doc
DOCS_TMP_DIR=`pwd`/.docs-staging
DOCS_OUT_DIR=docs

mkdir -p $DOCS_TMP_DIR

function build {
  baseURL="$1"
  if [ -z $baseURL ]; then
    baseURL="/mktree"
  fi

  workdir=$(mktemp -d)

  echo "info: copying doc/ to $workdir"
  cp -r doc $workdir


  echo "info: running hugo"
  pushd $workdir/doc
  hugo --baseURL=$baseURL # We serve at /mktree on github pages
  popd
  
  #
  # Post-build documentation cleanup.
  #

  echo "info: disabling github pages"
  touch $workdir/doc/public/.nojekyll 

  echo "info: post-processing hugo output"
  go run ./cmd/gendocs "$workdir/doc/public"

  # Finalization.
  echo "info: copying generated docs to $DOCS_OUT_DIR"
  rsync -av --progress --delete $workdir/doc/public/ $DOCS_OUT_DIR
  rm -rf $workdir
}

function serve {
  #pushd $DOCS_SRC_DIR
  #hugo server -D
  build "/"
  pushd $DOCS_OUT_DIR
  python3 -m http.server
  popd
}

while getopts "bps" COMMAND; do
  case $COMMAND in
  b) 
     build "/mktree"
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

