# This script ensures that GOPATH is suitably modified given the go version
# on our PATH. It works on both local machines (with smartcd) and on the CI
# server.
#
# It also ensures that GOBIN is suitably set and PATH updated to include GOBIN.

source "${BASH_SOURCE%/*}/commonFunctions.bash"

if [ "${LOADED_SETUP_GOPATH:-}" == "true" ]
then
	return
fi

autostash_or_export GOBIN="$(readlink -m "${BASH_SOURCE%/*}/../.bin")"

autostash_or_export PATH="$GOBIN:$PATH"

# Pre Go 1.11 check
# [[ "$(go version | cut -d ' ' -f 3)" =~ go1.(9|10).[0-9]+ ]]

if [ "${GO111MODULE:-}" != "on" ]
then
	autostash_or_export GOPATH="$(readlink -m "${BASH_SOURCE%/*}/../_vendor"):$GOPATH"
fi

LOADED_SETUP_GOPATH=true
