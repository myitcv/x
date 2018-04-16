#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

cd $(git rev-parse --show-toplevel)

$go install golang.org/x/tools/cmd/goimports

args="-l"
if [ $# -gt 0 ]
then
	args="$@"
fi

u=$(goimports $args **/!(gen_*).go)
u=$u$(gofmt $args **/gen_*.go)

if [ "$u" != "" ]
then
	echo "The following files are not formatted"
	echo ""
	echo "$u"
	exit 1
fi
