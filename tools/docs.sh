#!/bin/bash

set -eu

DEFAULT_BASE_URL="/mktree"
DEFAULT_DOCS_OUT_DIR="docs"

function build {
  BASE_URL="$1"
  DOCS_OUT_DIR="$2"

  if [ -z $BASE_URL ]; then
    BASE_URL=$DEFAULT_BASE_URL
  fi

  if [ -z DOCS_OUT_DIR ]; then
    DOCS_OUT_DIR=$DEFAULT_DOCS_OUT_DIR
  fi

  workdir=$(mktemp -d)

  echo "info: copying doc/ to $workdir"
  cp -r doc $workdir


  echo "info: running hugo"
  pushd $workdir/doc
  hugo --baseURL=$BASE_URL # We preview at /mktree on github pages
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

function preview {
  DOCS_OUT_DIR=$(mktemp -d)
  build "/" $DOCS_OUT_DIR
  pushd $DOCS_OUT_DIR
  python3 -m http.server
  popd
  rm -rf $DOCS_OUT_DIR
}

while getopts "bps" COMMAND; do
  case $COMMAND in
  b) 
     build "/mktree" "docs"
     ;;
  s) 
     preview
     ;;
  *)
     echo "Invalid option"
     exit 1
     ;;
  esac
done

