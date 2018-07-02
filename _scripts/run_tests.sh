#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "${BASH_SOURCE%/*}/common.bash"

go install myitcv.io/cmd/concsh

# work out a better way of priming the build tools
for i in cmd/pkgconcat
do
	pushd $i > /dev/null
	go install .
	popd > /dev/null
done

# TODO make sure we don't have nested run_tests.sh files

for i in $(find !(_scripts) -mindepth 1 -name run_tests.sh -exec bash -c 'dirname $(dirname {})' \;)
do
	cat <<EOD
bash -c "set -e; echo '---- $i'; cd $i; ./_scripts/run_tests.sh; echo '----'; echo ''"
EOD
done | concsh

# now test the rest

echo Protobuf Include $PROTOBUF_INCLUDE

go test $(go list ./... | grep -v -f <(for i in $(find !(_scripts) -mindepth 1 -name run_tests.sh ); do dirname $(dirname $i); done))

for i in $(go list -f "{{if eq .Name \"main\"}}{{.Dir}}{{end}}" ./...)
do
	pushd $i > /dev/null
	go install
	popd > /dev/null
done

_scripts/update_readmes.sh

if [ "${CI:-}" == "true" ]
then
	function verifyGoGet()
	{
		local pkg=$1
		echo "Verifying go get for $pkg"
		(
		cd `mktemp -d`
		export GOPATH=$PWD
		go get $pkg
		)
	}

verifyGoGet "myitcv.io/cmd/concsh"
fi
