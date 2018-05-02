#!/usr/bin/env bash

set -eu

target=$HOME/gotip

echo "Will install ${GOTIP_VERSION:0:10} to $target"

mkdir -p $target
cd $target
if [ ! -e .git ]
then
	git clone -q https://github.com/golang/go .
fi

git checkout -qf $GOTIP_VERSION
cd src
GOROOT_BOOTSTRAP=$(go env GOROOT) ./make.bash
