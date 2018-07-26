#!/usr/bin/env vbash

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

trap "go version" EXIT

only_run_on_ci_server

if [ "$(uname -m)" != "x86_64" ]
then
	echo "Unkown architecture"
	exit 1
fi

os=$(uname | tr '[:upper:]' '[:lower:]')
arch="amd64"

if [[ "$GO_VERSION" = go* ]]
then
	cd $HOME
	curl -sL  https://dl.google.com/go/$GO_VERSION.${os}-${arch}.tar.gz | tar -zx
else
	# tip
	source="https://s3-eu-west-1.amazonaws.com/io.myitcv.gobuilds/${os}_${arch}/$GO_VERSION.tar.gz"
	target=$HOME/go

	echo "Will install ${GO_VERSION:0:10} from $source to $target"

	mkdir -p $target
	cd $target

	curl -L -s $source | tar -xz
fi
