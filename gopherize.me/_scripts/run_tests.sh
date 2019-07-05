#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

# ensure we are in the right directory
cd "${BASH_SOURCE%/*}/.."

sub_git_files | gen_files | xargs rm -f

gobin -m -run myitcv.io/cmd/gogenerate $(subpackages)
go test $(subpackages)
go vet $(subpackages)
gobin -m -run myitcv.io/react/cmd/reactVet $(subpackages)

ensure_go_formatted $(sub_git_files | non_gen_go_files)
ensure_go_gen_formatted $(sub_git_files | gen_go_files)
