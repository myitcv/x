#!/usr/bin/env vbash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

# TODO: work out a better way of priming the build tools
go install myitcv.io/cmd/concsh myitcv.io/cmd/pkgconcat
go install golang.org/x/tools/cmd/goimports

# Top-level run_tests.sh only.
# check we don't have doubly-nested sub tests - we don't support this yet
diff -wu <(nested_test_dirs) <(nested_test_dirs | grep -v -f <(nested_test_dir_patterns))

# TODO for now we manually specify the run order of nested test dirs
# that is until we automate the dependency order (or use Bazel??)
nested_order="sorter
immutable
cmd/gjbt
react
gopherize.me"

# TODO remove when we revert back to running tests in parallel
diff -wu <(cat <<< "$nested_order" | sort) <(nested_test_dirs | sort)
for i in $nested_order
do
	pushd $i > /dev/null
	echo "---- $i"
	./_scripts/run_tests.sh
	echo "----"
	echo ""
	popd > /dev/null
done

# TODO re-enable this once we correctly calculate the dependency graph
# and only run things in parallel where we can
#
# run_nested_tests

go generate $(subpackages)

ensure_go_formatted $(sub_git_files | grep -v '^_vendor/' | non_gen_go_files)
ensure_go_gen_formatted $(sub_git_files | grep -v '^_vendor/' | gen_go_files)

go test $(subpackages)

install_main_go $(subpackages)

# TODO remove once we have Go 1.11
install_deps $(subpackages)

go vet $(subpackages)

_scripts/update_readmes.sh

if [ $(running_on_ci_server) == "yes" ]
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
