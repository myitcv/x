# This script ensures that GOPATH is suitably modified given the go version
# on our PATH. It works on both local machines (with smartcd) and on the CI
# server.
#
# It also ensures that GOBIN is suitably set and PATH updated to include GOBIN.

if [ "${LOADED_SETUP_GOPATH:-}" == "true" ]
then
	return
fi

if [ "$(type -t autostash || true)" == "function" ]
then
	autostash GOBIN="$(readlink -m "${BASH_SOURCE%/*}/../.bin")"
else
	export GOBIN="$(readlink -m "${BASH_SOURCE%/*}/../.bin")"
fi

if [[ "$(go version | cut -d ' ' -f 3)" =~ go1.(9|10).[0-9]+ ]]
then
	if [ "$(type -t autostash || true)" == "function" ]
	then
		autostash GOPATH="$(readlink -m "${BASH_SOURCE%/*}/../_vendor"):$GOPATH"
	else
		export GOPATH="$(readlink -m "${BASH_SOURCE%/*}/../_vendor"):$GOPATH"
	fi
fi

LOADED_SETUP_GOPATH=true
