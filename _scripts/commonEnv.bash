# If our environment hasn't set these variables, pick up the default "pinned"
# version. This allows us to test a multitude of different protobuf versions
# via CI

if [ "${LOADED_COMMON_ENV:-}" == "true" ]
then
	return
fi

# We deliberately do NOT source commonFunctions here because this file is
# (transitively) sourced from a user's shell
#
# source "${BASH_SOURCE%/*}/commonFunctions.bash"
#

# Some CI-only env setup

if [ $(running_on_ci_server) == "yes" ]
then
	# TODO ensure the go build cache is also cache
	export CI_CACHE_DIR=~/cache
	export CI_DEPENDENCIES_DIR=$CI_CACHE_DIR/depedencies

	export PROTOBUF_INSTALL_DIR=$CI_DEPENDENCIES_DIR/protobuf

	export CHROMEDRIVER_INSTALL_DIR=$CI_DEPENDENCIES_DIR/chromedriver
fi

if [ "${PROTOBUF_VERSION:-}" == "" ]
then
	autostash_or_export PROTOBUF_VERSION="$(cat "${BASH_SOURCE%/*}/../.dependencies/protobuf_version")"
fi
if [ "${CHROMEDRIVER_VERSION:-}" == "" ]
then
	autostash_or_export CHROMEDRIVER_VERSION="$(cat "${BASH_SOURCE%/*}/../.dependencies/chromedriver_version")"
fi

if [ $(running_on_ci_server) != "yes" ]
then
	autostash_or_export PROTOBUF_INCLUDE="$PROTOBUF_INSTALL_DIR/$PROTOBUF_VERSION/include"
fi

autostash_or_export LOADED_COMMON_ENV=true
