#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

if [ ! -z "$(git status --porcelain)" ]
then
  echo "Git is not clean"
  git status
  git diff
  exit 1
fi
