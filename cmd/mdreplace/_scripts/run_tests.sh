#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

rm -f !(_vendor)/**/gen_*.go

$go install golang.org/x/tools/cmd/stringer

$go generate ./...
$go test ./...

# we can remove this once we resolve https://github.com/golang/go/issues/24661
$go install ./...
