#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

cd "${BASH_SOURCE%/*}/.."

gobin -m -run myitcv.io/cmd/gogenerate $(subpackages) ./cmd/immutableVet/_testFiles
go test $(subpackages)
go vet $(subpackages)
gobin -m -run myitcv.io/immutable/cmd/immutableVet myitcv.io/immutable/example

ensure_go_formatted $(sub_git_files | non_gen_go_files)
ensure_go_gen_formatted $(sub_git_files | gen_go_files)
