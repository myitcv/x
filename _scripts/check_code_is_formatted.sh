#!/usr/bin/env vbash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

cd $(git rev-parse --show-toplevel)

go install golang.org/x/tools/cmd/goimports

# in case we don't have any matching files in the globs below
shopt -s nullglob

z=$(goimports -l $(git ls-files | grep -v '^_vendor' | grep -v '^react/_talks' | grep '.go$' | grep -v '/gen_[^/]*$'))
if [ ! -z "$z" ]
then
	echo "The following files are not formatted:"
	echo ""
	echo "$z"
	exit 1
fi

z=$(gofmt -l $(git ls-files | grep -v '^_vendor' | grep -v '^react/_talks' | grep '/gen_[^/]*.go$'))

if [ ! -z "$z" ]
then
	echo "The following generated files are not formatted:"
	echo ""
	echo "$z"
	exit 1
fi
