#!/usr/bin/env bash

set -euo pipefail

cd "${BASH_SOURCE%/*}/../"

proxy=""
if [ "${CI:-}" != "true" ]
then
	proxy="-v $GOPATH/pkg/mod/cache/download:/cache -e GOPROXY=file:///cache"
fi

# We use TRAVIS=true to simulate the fact that we are on the CI server (because
# this is what is used by running_on_ci_server). But we use CI=true to distinguish
# whether we are running in Docker locally or not, because in a couple of instances
# we need that distinction. So CI=true is _not_ set here.
export TRAVIS="${TRAVIS:-true}"

export GO_VERSION="${GO_VERSION:-$(go version | awk '{print $3}')}"

echo docker run $proxy --env-file ./.docker_env_file -v $PWD:/home/$USER/x -w /home/$USER/x --rm x_monorepo ./_scripts/docker_run.sh
docker run $proxy --env-file ./.docker_env_file -v $PWD:/home/$USER/x -w /home/$USER/x -v=/var/run/docker.sock:/var/run/docker.sock -v=/tmp:/tmp --rm x_monorepo ./_scripts/docker_run.sh
