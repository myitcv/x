#!/usr/bin/env vbash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

if [[ ! "$(goVersion)" =~ go1.(9|10).[0-9]+ ]]
then
	go mod tidy
	go list all > /dev/null
fi

for i in $(ls _scripts/known_diffs)
do
	echo "goVersion $(goVersion)"
	if [[ "$(goVersion)" = $i* ]]
	then
		for j in $(find _scripts/known_diffs/$i -type f)
		do
			git apply $j
		done
	fi
done

if [ ! -z "$(git status --porcelain)" ]
then
  echo "Git is not clean"
  git status
  git diff
  exit 1
fi
