#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

function error() {
  local lineno="$1"
  local file="$2"

  # intentional so we can test BASH_SOURCE
  if [[ -n "$file" ]] ; then
    echo "Error on line $file:$lineno"
  fi

  exit 1
}

trap 'set +u; error "${LINENO}" "${BASH_SOURCE}"' ERR

set -u
set -v
shopt -s globstar
shopt -s extglob

rm -f **/gen_*.go

go install myitcv.io/immutable/cmd/immutableGen
go install myitcv.io/immutable/cmd/immutableVet

pushd cmd/immutableVet/_testFiles
go generate
popd

go generate ./...
go install ./...
go vet ./...
go test ./...
immutableVet myitcv.io/immutable/example

