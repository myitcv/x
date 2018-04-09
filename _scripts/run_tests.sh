#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

# install vgo on CI server
if [ "${CI:-}" == "true" ]
then
	go get -u golang.org/x/vgo
	pushd $(go list -f "{{.Dir}}" golang.org/x/vgo) > /dev/null
	git checkout -qf $VGO_COMMIT
	go install
	popd > /dev/null

	# so we can access Github without hitting rate limits
	echo "machine api.github.com login $GH_USER password $GH_TOKEN" >> $HOME/.netrc

	# now setup our cache for ensuring integrity of CI builds
	pushd $GOPATH > /dev/null

	git clone -q https://github.com/myitcv/cachex

	cd cachex

	for i in $(find -name *.mod)
	do
		d=$(dirname $i)
		v=$(basename $i .mod)
		echo "$v" >> "$d/list"
	done

	git clone -q https://github.com/myitcv/pubx /tmp/pubx
	mv /tmp/pubx/myitcv.io ./myitcv.io

	export GOPROXY="file://$PWD"

	popd > /dev/null
fi

export PATH=$GOPATH/bin:$PATH

$go version
$go env

# can potentially go when we get a resolution on
# https://github.com/golang/go/issues/24748
echo "GOPROXY=\"${GOPROXY:-}\""

# get all packages that do not belong to a module that has its
# own _scripts/run_tests.sh file
for i in $(find -mindepth 2 -iname go.mod -exec dirname '{}' \;)
do
	echo "---- $i"
	pushd $i > /dev/null
	if [ -f ./_scripts/run_tests.sh ]
	then
		./_scripts/run_tests.sh
	else
		$go generate ./...
		$go test ./...

		# we can remove this once we resolve https://github.com/golang/go/issues/24661
		$go install ./...
	fi
	popd > /dev/null
	echo "----"
	echo ""
done

echo Checking markdown files are current
# by this point we will have mdreplace installed. Hence check that
# committed .md files are "fresh"
mdreplace -w **/*.md
