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

if [[ "$(goVersion)" =~ go1.(9|10).[0-9]+ ]]
then
	autostash_or_export GOPATH="$(readlink -m "${BASH_SOURCE%/*}/../_vendor"):${BASH_SOURCE%/*}/../../../"
fi

if [ $(running_on_ci_server) == "yes" ]
then
	export GO111ROOT="$HOME/go111/go"

fi

autostash_or_export LOADED_SETUP_GOPATH=true

