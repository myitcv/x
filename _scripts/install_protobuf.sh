#!/usr/bin/env bash

source "${BASH_SOURCE%/*}/common.bash"

# Installs protobuf to $PROTOBUF_INSTALL_DIR.

# TODO make work on different platforms. Only works on Linux for now.

if [ -e $PROTOBUF_INSTALL_DIR/$PROTOBUF_VERSION/bin/protoc ]
then
	# nothing to do
	exit
fi

DOWNLOAD_URL=https://github.com/google/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-linux-x86_64.zip

tf="$(mktemp).zip"
trap "rm -f $tf" EXIT

curl -sL $DOWNLOAD_URL > $tf

mkdir -p $PROTOBUF_INSTALL_DIR/$PROTOBUF_VERSION

cd $PROTOBUF_INSTALL_DIR/$PROTOBUF_VERSION > /dev/null

unzip -q $tf
