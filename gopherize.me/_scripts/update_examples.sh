#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "${BASH_SOURCE%/*}/common.bash"

r=$HOME/.cache/gopherize.me_site
t=$(mktemp -d)

(
	cd $r

	echo "Fetching https://github.com/myitcv/gopherize.me_site into $r"

	git fetch
	git checkout -f master
	git reset --hard origin/master
	rm -rf $r/*
)

echo ""

echo "Copying..."

(
	cd $t
	wget --quiet --mirror http://localhost:8081/myitcv.io/gopherize.me/client/
)

cp -rp $t/localhost:8081/myitcv.io/gopherize.me/client/* $r

du -sh $r/!(artwork)

cp -rp artwork $r

echo ""

cd $r
git config hooks.stopbinaries false

if [ -z "$(git status --porcelain)" ]
then
	echo "No changes to commit"
	exit 0
fi

git add -A
git commit -am "Examples update at $(date)"
git push -f
