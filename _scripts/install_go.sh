#!/usr/bin/env bash

source "${BASH_SOURCE%/*}/common.bash"

trap "go version" EXIT

if [ "${USE_GO_TIP:-}" != "true" ]
then
	# nothing to do
	exit
fi

source="https://github.com/myitcv/gobuilds/raw/master/linux_amd64/$GO_TIP_VERSION.tar.gz"
target=$HOME/gotip

echo "Will install ${GO_TIP_VERSION:0:10} from $source to $target"

mkdir -p $target
cd $target

curl -L -s $source | tar -xz
