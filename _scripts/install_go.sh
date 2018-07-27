#!/usr/bin/env vbash

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

trap "go version" EXIT

only_run_on_ci_server

if [ "$(uname -m)" != "x86_64" ]
then
	echo "Unkown architecture"
	exit 1
fi

tf=$(mktemp)
os=$(uname | tr '[:upper:]' '[:lower:]')
arch="amd64"

if [[ "$GO_VERSION" = go* ]]
then
	source="https://dl.google.com/go/$GO_VERSION.${os}-${arch}.tar.gz"
	curl -sL $source > $tf
else
	source="s3://io.myitcv.gobuilds/${os}_${arch}/$GO_VERSION.tar.gz"
	aws s3 cp $source $tf
fi

echo "Will install ${GO_VERSION} from $source"

cd $HOME
tar -zxf $tf
