#!/usr/bin/env vbash

set -euo pipefail

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

cd "${BASH_SOURCE%/*}/../"

# Because we use beta and stable below, use the value of $(google-chrome --version) on the
# host machine (rebuilding the image) as an indicator of whether to rebuild or not
export CHROME_VERSION="$(google-chrome --version)"
export CHROMEDRIVER_77_VERSION=77.0.3865.40
export CHROMEDRIVER_76_VERSION=76.0.3809.126
export CHROMEDRIVER_75_VERSION=75.0.3770.140

for g in 1.12.9 1.11.13
do
	for i in beta stable
	do
		docker build --build-arg PROTOBUF_VERSION=$PROTOBUF_VERSION --build-arg CHROME_VERSION="$CHROME_VERSION" --build-arg GO_VERSION=go$g --build-arg CHROMEDRIVER_77_VERSION=$CHROMEDRIVER_77_VERSION --build-arg CHROMEDRIVER_76_VERSION=$CHROMEDRIVER_76_VERSION --build-arg CHROMEDRIVER_75_VERSION=$CHROMEDRIVER_75_VERSION --build-arg CHROME_CHANNEL=$i --build-arg "VBASHPATH=$(realpath --relative-to=$PWD $(gobin -m -p github.com/myitcv/vbash))" --build-arg "GOBINPATH=$(realpath --relative-to=$PWD $(gobin -m -p github.com/myitcv/gobin))" -t myitcv/x_monorepo:chrome_${i}_go${g} . ##
	done
done
