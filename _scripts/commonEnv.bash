# If our environment hasn't set these variables, pick up the default "pinned"
# version. This allows us to test a multitude of different protobuf versions
# via CI

autostash_or_export()
{
	if [ "$(type -t autostash || true)" == "function" ]
	then
		autostash "$@"
	else
		export "$@"
	fi
}

if [ "${PROTOBUF_VERSION:-}" == "" ]
then
	autostash_or_export PROTOBUF_VERSION="$(cat .dependencies/protobuf_version)"
fi
if [ "${CHROMEDRIVER_VERSION:-}" == "" ]
then
	autostash_or_export CHROMEDRIVER_VERSION="$(cat .dependencies/chromedriver_version)"
fi

autostash_or_export PROTOBUF_INCLUDE="$PROTOBUF_INSTALL_DIR/$PROTOBUF_VERSION/include"
