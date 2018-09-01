#!/usr/bin/env vbash

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

only_run_on_ci_server

# Ensure cache directories etc exist

mkdir -p $CI_CACHE_DIR
mkdir -p $CI_DEPENDENCIES_DIR
mkdir -p $PROTOBUF_INSTALL_DIR

# Install protobuf and other external deps

"${BASH_SOURCE%/*}/install_protobuf.sh"
"${BASH_SOURCE%/*}/install_chromedriver.sh"

mkdir -p $GOBIN

git clone --depth=1 https://github.com/myitcv/cachex $HOME/cachex

if [[ "$GO_VERSION" = go1.10* ]]
then
	mkdir $HOME/go111
	installGo $GO_111VERSION $HOME/go111

	mkdir -p _vendor

	export GO111MODULE=on
	go=$HOME/go111/go/bin/go
	$go mod edit -replace github.com/gopherjs/gopherjs=github.com/myitcv/gopherjs@v0.0.0-20180708170036-38b413be4187
	$go mod vendor
	unset GO111MODULE

	git checkout go.mod go.sum

	if [ -e _vendor/src ]
	then
		echo "_vendor/src exists; why?"
		exit 1
	fi

	mv vendor _vendor/src
fi
