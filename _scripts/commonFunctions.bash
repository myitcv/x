if [ "${LOADED_COMMON_FUNCTIONS:-}" == "true" ]
then
	return
fi

autostash_or_export()
{
	if [ "$(type -t autostash || true)" == "function" ]
	then
		autostash "$@"
	else
		export "$@"
	fi
}
export -f autostash_or_export

running_on_ci_server()
{
	local res
	if [ "${TRAVIS:-}" == "true" ]
	then
		res=yes
	else
		res=no
	fi
	echo $res
}
export -f running_on_ci_server

only_run_on_ci_server()
{
	if [ $(running_on_ci_server) != "yes" ]
	then
		echo "This script can ONLY be run on the CI server"
		exit 1
	fi
}
export -f only_run_on_ci_server

cwd_as_import_path()
{
	local i
	for i in $(sed -e "s/:/\n/g" <<< "$GOPATH")
	do
		if [[ "$PWD" =~ $i* ]]
		then
			echo ${PWD#${i}/src/}
			return
		fi
	done

	echo "could not resolve $PWD to import path"
	exit 1
}
export -f cwd_as_import_path

subpackages()
{
	local ip=$(cwd_as_import_path)

	go list ./... | ( grep -v -f <(sed -e 's+^\(.*\)$+^myitcv.io/\1/+' <<< "$(nested_test_dirs)") || true )
}
export -f subpackages

nested_test_dirs()
{
	for i in $(find -path ./_scripts -prune -o -name run_tests.sh -printf '%P\n')
	do
		dirname $(dirname "$i")
	done
}
export -f nested_test_dirs

nested_test_dir_patterns()
{
	nested_test_dirs | sed -e 's+^\(.*\)$+^\1/$+'
}
export -f nested_test_dir_patterns

sub_git_files()
{
	git ls-files | ( grep -v -f <(sed -e 's+^\(.*\)$+^\1/.*$+' <<< "$(nested_test_dirs)") || true )
}
export -f sub_git_files

ensure_go_formatted()
{
	if [ "$#" == "0" ]
	then
		return
	fi
	local z=$(goimports -l "$@")
	if [ ! -z "$z" ]
	then
		echo "The following files are not formatted:"
		echo ""
		echo "$z"
		exit 1
	fi
}
export -f ensure_go_formatted

ensure_go_gen_formatted()
{
	if [ "$#" == "0" ]
	then
		return
	fi
	local z=$(gofmt -l "$@")
	if [ ! -z "$z" ]
	then
		echo "The following generated files are not formatted:"
		echo ""
		echo "$z"
		exit 1
	fi
}
export -f ensure_go_gen_formatted

gen_files()
{
	grep '\(^gen_\|/gen_\)[^/]\+$' || true
}
export -f gen_files

non_gen_files()
{
	grep -v '\(^gen_\|/gen_\)[^/]\+$' || true
}
export -f non_gen_files

go_files()
{
	grep '/\?[^/]\+.go$' || true
}
export -f go_files

non_gen_go_files()
{
	go_files | non_gen_files
}
export -f non_gen_go_files

gen_go_files()
{
	go_files | gen_files
}
export -f gen_go_files

run_nested_tests()
{
	for i in $(nested_test_dirs)
	do
		cat <<EOD
bash -c "set -e; echo '---- $i'; cd $i; ./_scripts/run_tests.sh; echo '----'; echo ''"
EOD
	done | concsh
}
export -f run_nested_tests

install_main_go()
{
	for i in $(go list -f "{{if eq .Name \"main\"}}{{.Dir}}{{end}}" "$@")
	do
		pushd $i > /dev/null
		go install
		popd > /dev/null
	done
}
export -f install_main_go

# workaround for Go 1.10 pre export data in Go 1.11
install_deps()
{
	go list -f "{{ range .Deps}}{{.}}
	{{end}}" "$@" | xargs go install
}
export -f install_deps

verifyGoGet()
{
	local pkg=$1
	echo "Verifying go get for $pkg"
	(
	cd `mktemp -d`
	export GOPATH=$PWD
	GO111MODULE=auto
	go env
	go get $pkg
	)
}
export -f verifyGoGet

installGo() {
	# takes a two argument
	#
	# 1. go version
	# 2. the target directory into which we will install the go directory
	#
	tf=$(mktemp)
	os=$(uname | tr '[:upper:]' '[:lower:]')
	arch="amd64"

	if [[ "$1" = go* ]]
	then
		source="https://dl.google.com/go/$1.${os}-${arch}.tar.gz"
		curl -sL $source > $tf
	else
		source="s3://io.myitcv.gobuilds/${os}_${arch}/$1.tar.gz"
		aws s3 cp $source $tf
	fi

	echo "Will install ${1} from $source to $2"
	tar -C $2 -zxf $tf
}
export -f installGo

# **********************
LOADED_COMMON_FUNCTIONS=true
