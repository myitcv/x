#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "${BASH_SOURCE%/*}/common.bash"

if [ "${CI:-}" == "true" ]
then
	go get golang.org/x/tools/cmd/goimports
fi

# ensure we are in the right directory
cd "${BASH_SOURCE%/*}/.."

find -path ./_vendor -prune -o -name "gen_*.go" -exec rm '{}' \;

go generate ./...

if compgen -G "!(_vendor|_talks)/**/!(gen_*).go !(gen_*).go)"
then
	z=$(goimports -l !(_vendor|_talks)/**/!(gen_*).go !(gen_*).go)
	if [ ! -z "$z" ]
	then
		echo "The following files are not formatted:"
		echo ""
		echo "$z"
		exit 1
	fi
fi

if compgen -G "!(_vendor)/**/gen_*.go gen_*.go"
then
	z=$(gofmt -l !(_vendor)/**/gen_*.go gen_*.go)

	if [ ! -z "$z" ]
	then
		echo "The following generated files are not formatted:"
		echo ""
		echo "$z"
		exit 1
	fi
fi

go test ./...
