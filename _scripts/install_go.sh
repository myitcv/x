#!/usr/bin/env bash

source "${BASH_SOURCE%/*}/common.bash"

target=$HOME/gotip

echo "Will install ${GOTIP_VERSION:0:10} from $GOTIP_REPO to $target"

mkdir -p $target
cd $target

if [ ! -e .git ]
then
	echo cloning
	git clone -q $GOTIP_REPO .
else
	echo fetching
	git fetch -q $GOTIP_REPO
fi

git checkout -qf $GOTIP_VERSION
cd src
GOROOT_BOOTSTRAP=$(go env GOROOT) ./make.bash
