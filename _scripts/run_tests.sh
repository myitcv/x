#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

# install vgo on CI server
if [ "${CI:-}" == "true" ]
then
	go get -u golang.org/x/vgo
	pushd $(go list -f "{{.Dir}}" golang.org/x/vgo) > /dev/null

	# git checkout -qf $VGO_COMMIT

	# override from deplist CL
	git fetch -q https://go.googlesource.com/vgo refs/changes/55/105855/4 && git checkout -qf FETCH_HEAD
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

	$go version
	$go env

	# can potentially go when we get a resolution on
	# https://github.com/golang/go/issues/24748
	echo "GOPROXY=\"${GOPROXY:-}\""
fi

export PATH=$GOPATH/bin:$PATH

# work out a better way of priming the build tools
for i in cmd/pkgconcat cmd/gg
do
	pushd $i > /dev/null
	$go install .
	popd > /dev/null
done

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
		if [ -f ./_scripts/pre_run_tests.sh ]
		then
			./_scripts/pre_run_tests.sh
		fi

		$gg ./...
		$go test ./...

		if [ -f ./_scripts/post_run_tests.sh ]
		then
			./_scripts/post_run_tests.sh
		fi
	fi
	popd > /dev/null
	echo "----"
	echo ""
done

# we use regular go to list here because of https://github.com/golang/go/issues/24749;
# this is also the reason why we need to change to the directory to do the vgo install
for i in $(go list -f "{{if eq .Name \"main\"}}{{.Dir}}{{end}}" ./...)
do
	pushd $i > /dev/null
	$go install
	popd > /dev/null
done

_scripts/update_readmes.sh
