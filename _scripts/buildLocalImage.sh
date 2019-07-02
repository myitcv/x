#!/usr/bin/env bash

set -euo pipefail

cd "${BASH_SOURCE%/*}/.."

export "CHROME_CHANNEL=${CHROME_CHANNEL:-beta}"
export "GO_VERSION=${GO_VERSION:-$(go version | awk '{print $3}')}"

cat Dockerfile.user \
	| envsubst '$GO_VERSION,$CHROME_CHANNEL' \
	| docker build -t x_monorepo --build-arg USER=$USER --build-arg UID=$UID --build-arg GID=$(id -g $USER) -
