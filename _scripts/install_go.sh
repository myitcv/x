#!/usr/bin/env vbash

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

trap "go version" EXIT

only_run_on_ci_server

if [ "$(uname -m)" != "x86_64" ]
then
	echo "Unkown architecture"
	exit 1
fi

installGo $GO_VERSION $HOME
