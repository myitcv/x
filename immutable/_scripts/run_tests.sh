#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

cd "${BASH_SOURCE%/*}/.."

go install myitcv.io/immutable/cmd/immutableGen myitcv.io/immutable/cmd/immutableVet

gobin -m -run myitcv.io/cmd/gg $(subpackages) ./cmd/immutableVet/_testFiles
go install $(subpackages)
go test $(subpackages)
go vet $(subpackages)
immutableVet myitcv.io/immutable/example

ensure_go_formatted $(sub_git_files | non_gen_go_files)
ensure_go_gen_formatted $(sub_git_files | gen_go_files)
