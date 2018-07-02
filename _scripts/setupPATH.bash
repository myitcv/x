# This script ensures that our PATH is correctly setup. Can be sourced either
# locally or on the CI server

if [ "${LOADED_SETUP_PATH:-}" == "true" ]
then
	return
fi

source "${BASH_SOURCE%/*}/commonEnv.bash"

if [ "$(type -t autostash || true)" == "function" ]
then
	autostash PATH="$(readlink -m "${BASH_SOURCE%/*}/../.bin"):$CHROMEDRIVER_INSTALL_DIR/$CHROMEDRIVER_VERSION:$PROTOBUF_INSTALL_DIR/$PROTOBUF_VERSION/bin:$PATH"
else
	export PATH="$(readlink -m "${BASH_SOURCE%/*}/../.bin"):$CHROMEDRIVER_INSTALL_DIR/$CHROMEDRIVER_VERSION:$PROTOBUF_INSTALL_DIR/$PROTOBUF_VERSION/bin:$PATH"
fi

LOADED_SETUP_PATH=true
