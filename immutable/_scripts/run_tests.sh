#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

rm -f !(_vendor)/**/gen_*.go

for i in ./cmd/immutableGen ./cmd/immutableVet
do
	pushd $i > /dev/null
	$gg
	$go install
	popd > /dev/null
done

# this step is needed because _testFiles is not walked by ./...
pushd cmd/immutableVet/_testFiles
$gg
popd

$gg ./...
$go test ./...

$go install myitcv.io/immutable/cmd/immutableVet

immutableVet myitcv.io/immutable/example
