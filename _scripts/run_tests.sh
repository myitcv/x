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
	echo "=============================" #!
	echo "$i/_scripts/run_tests.sh" #!
	pushd $i > /dev/null
	./_scripts/run_tests.sh
	popd > /dev/null
done

# TODO come up with a better way of doing mutli-OS-ARCH stuff
GOOS=linux GOARCH=amd64 gobin -m -run myitcv.io/cmd/gg myitcv.io/cmd/protoc
GOOS=darwin GOARCH=amd64 gobin -m -run myitcv.io/cmd/gg myitcv.io/cmd/protoc

for i in $(find !(_vendor) -name go.mod -execdir pwd \;)
do
	echo "=============================" #!
	echo "$i: regular run" #!
	pushd $i > /dev/null

	gobin -m -run myitcv.io/cmd/gg $(subpackages)

	ensure_go_formatted $(sub_git_files | grep -v '^_vendor/' | non_gen_go_files)
	ensure_go_gen_formatted $(sub_git_files | grep -v '^_vendor/' | gen_go_files)

	go test $(subpackages)

	install_main_go $(subpackages | grep -v myitcv.io/cmd/gg/internal/go)

	go vet $(subpackages)

	go mod tidy
	go list all > /dev/null

	popd > /dev/null
done

./_scripts/update_readmes.sh

if [ $(running_on_ci_server) == "yes" ]
then
	verifyGoGet myitcv.io/cmd/concsh
fi
