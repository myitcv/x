#!/usr/bin/env vbash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

node --version
npm --version
google-chrome --version

# ensure we are in the right directory
cd "${BASH_SOURCE%/*}/.."

./_scripts/webpack_deps.sh

for i in $(cat .vendored_bin_deps .bin_deps)
do
	go install $i
done

sub_git_files | gen_files | xargs rm -f

{
	pushd examples/sites/helloworld

	rm -f *.{go,html}
	reactGen -init minimal

	popd
}

{
	pushd examples/sites/helloworldbootstrap

	rm -f *.{go,html}
	reactGen -init bootstrap

	popd
}

go generate $(subpackages)

install_main_go $(subpackages)

# TODO remove once we have Go 1.11
install_deps $(subpackages)

# we install the deps above because one of the reactVet tests
# requires the deps to be present
go test $(subpackages)

# TODO work out a better way of excluding the cmd packages
# or making them exclude themselves by virtue of a build tag
gjbt $(subpackages | grep -v 'myitcv.io/react/cmd/')

go vet $(subpackages)
reactVet $(subpackages)
immutableVet $(subpackages)

# We need to explicitly test the generated test files
# because these are not found by go list
go generate myitcv.io/react/cmd/stateGen/_testFiles/
go test myitcv.io/react/cmd/stateGen/_testFiles/

ensure_go_formatted $(sub_git_files | grep -v '^_talks/' | non_gen_go_files)
ensure_go_gen_formatted $(sub_git_files | gen_go_files)

if [ $(running_on_ci_server) == "yes" ]
then
	# off the back of https://github.com/myitcv/react/issues/116#issuecomment-380280847
	# ensure that we can go get myitcv.io/react/... without _vendor
	verifyGoGet myitcv.io/react/...
fi
