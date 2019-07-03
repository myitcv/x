#!/usr/bin/env vbash

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

only_run_on_ci_server

# Ensure cache directories etc exist

mkdir -p $CI_CACHE_DIR
mkdir -p $CI_DEPENDENCIES_DIR
mkdir -p $PROTOBUF_INSTALL_DIR

mkdir -p $GOBIN

if [ "${GOPROXY:-}" == "" ]
then
	git clone --depth=1 https://github.com/myitcv/cachex $HOME/cachex
	export GOPROXY=file://$HOME/cachex
fi

if [[ "$GO_VERSION" = go1.11* ]]
then
	go mod edit -replace github.com/gopherjs/gopherjs=github.com/myitcv/gopherjs@v1.11.50
fi
