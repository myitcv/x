#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source $(git rev-parse --show-toplevel)/_scripts/common.bash

# install vgo on CI server
if [ "${CI:-}" == "true" ]
then
	go get -u golang.org/x/vgo
	pushd $(go list -f "{{.Dir}}" golang.org/x/vgo)
	git checkout -f 890b798475a0fc2108fa88d9b2810d5f768f5752
	popd

	# so we can access Github without hitting rate limits
	echo "machine api.github.com login $GH_USER password $GH_TOKEN" >> $HOME/.netrc
fi

export PATH=$GOPATH/bin:$PATH


# get all packages that do not belong to a module that has its
# own _scripts/run_tests.sh file
for i in $(find -mindepth 2 -iname go.mod -exec dirname '{}' \;)
do
	pushd $i > /dev/null
	if [ -f $i/_scripts/run_tests.sh ]
	then
		./_scripts/run_tests.sh
	else
		# run the standard tests
		vgo generate ./...

		vgo test ./...
	fi
	popd > /dev/null
done
