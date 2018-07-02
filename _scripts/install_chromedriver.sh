#!/usr/bin/env bash

source "${BASH_SOURCE%/*}/common.bash"

# Installs chromedriver to $CHROMEDRIVER_INSTALL_DIR/$CHROMEDRIVER_VERSION
# (unless $CHROMEDRIVER_INSTALL_DIR/$CHROMEDRIVER_VERSION/chromedriver exists)

# TODO make work on different platforms. Only works on Linux for now.

if [ -e $CHROMEDRIVER_INSTALL_DIR/$CHROMEDRIVER_VERSION/chromedriver ]
then
	# nothing to do
	exit
fi

DOWNLOAD_URL=https://chromedriver.storage.googleapis.com/$CHROMEDRIVER_VERSION/chromedriver_linux64.zip

tf="$(mktemp).zip"
trap "rm -f $tf" EXIT

curl -s $DOWNLOAD_URL > $tf

mkdir -p $CHROMEDRIVER_INSTALL_DIR/$CHROMEDRIVER_VERSION

cd $CHROMEDRIVER_INSTALL_DIR/$CHROMEDRIVER_VERSION > /dev/null

unzip -q $tf
