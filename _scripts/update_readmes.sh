#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

args=""

if [ $(running_on_ci_server) == "yes" ]
then
	echo Checking markdown files are current
else
	echo Updating markdown files
fi


if [ $(running_on_ci_server) == "yes" ] || [ "${1:-}" == "-f" ]
then
	args="-long -online"
fi

# by this point we will have mdreplace installed. Hence check that
# committed .md files are "fresh"
mdreplace $args -w $(git ls-files !(_vendor)/**/*.md *.md)
