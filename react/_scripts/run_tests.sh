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

sub_git_files | gen_files | xargs rm -f

{
	pushd examples/sites/helloworld

	rm -f *.{go,html}
	gobin -m -run myitcv.io/react/cmd/reactGen -init minimal

	popd
}

{
	pushd examples/sites/helloworldbootstrap

	rm -f *.{go,html}
	gobin -m -run myitcv.io/react/cmd/reactGen -init bootstrap

	popd
}

go generate $(subpackages)
go test $(subpackages)

# TODO work out a better way of excluding the cmd packages
# or making them exclude themselves by virtue of a build tag
gobin -m -run myitcv.io/cmd/gjbt $(subpackages | grep -v 'myitcv.io/react/cmd/')

go vet $(subpackages)
gobin -m -run myitcv.io/react/cmd/reactVet $(subpackages)
gobin -m -run myitcv.io/immutable/cmd/immutableVet $(subpackages)

# We need to explicitly test the generated test files
# because these are not found by go list
go generate myitcv.io/react/cmd/stateGen/_testFiles/
go test myitcv.io/react/cmd/stateGen/_testFiles/

ensure_go_formatted $(sub_git_files | grep -v '^_talks/' | non_gen_go_files)
ensure_go_gen_formatted $(sub_git_files | gen_go_files)
