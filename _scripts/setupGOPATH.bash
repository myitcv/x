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

if [ $(running_on_ci_server) == "yes" ]
then
	export GO111ROOT="$HOME/go111/go"

fi

autostash_or_export LOADED_SETUP_GOPATH=true

