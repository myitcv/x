#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

rm -f $(sub_git_files | gen_files)

# TODO use gg
go install myitcv.io/sorter/cmd/sortGen
go install myitcv.io/immutable/cmd/immutableGen

gobin -m -run myitcv.io/cmd/gg $(subpackages)
go install $(subpackages)
go vet $(subpackages)
go test $(subpackages)

pushd cmd/sortGen/_testFiles/ > /dev/null

gobin -m -run myitcv.io/cmd/gg $(subpackages)
go test $(subpackages)
go install $(subpackages)
go vet $(subpackages)

popd > /dev/null

ensure_go_formatted $(sub_git_files | non_gen_go_files)
ensure_go_gen_formatted $(sub_git_files | gen_go_files)
