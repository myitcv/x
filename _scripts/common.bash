# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

if [ "${LOADED_COMMON_BASH:-}" == "true" ]
then
	return
fi

set -u
set -o pipefail

# The following must be set _before_ the trap in order that the trap also applies within
# function bodies
#
# See https://www.gnu.org/software/bash/manual/html_node/Shell-Functions.html#Shell-Functions
set -o errtrace

# We assume that PATH will have been setup in such a way as to leave
# the correct version of go in place. This allows us to test multiple
# versions of Go via CI or to "pin" via go version, e.g. the logic that
# sets up our _vendor GOPATH or not:

source "${BASH_SOURCE%/*}/setupGOPATH.bash"

shopt -s globstar
shopt -s extglob

error() {
	local lineno="$1"
	local file="$2"

  # intentional so we can test BASH_SOURCE
  if [[ -n "$file" ]] ; then
	  echo "Error on line $file:$lineno"
  fi

  exit 1
}

trap 'set +u; error "${LINENO}" "${BASH_SOURCE}"' ERR

running_on_ci_server()
{
	set +u
	local res
	if [ "$TRAVIS" == "true" ]
	then
		res=yes
	else
		res=no
	fi
	set -u
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

# Some CI-only env setup

if [ $(running_on_ci_server) == "yes" ]
then
	# TODO ensure the go build cache is also cache
	export CI_CACHE_DIR=~/cache
	export CI_DEPENDENCIES_DIR=$CI_CACHE_DIR/depedencies

	export PROTOBUF_INSTALL_DIR=$CI_DEPENDENCIES_DIR/protobuf

	export CHROMEDRIVER_INSTALL_DIR=$CI_DEPENDENCIES_DIR/chromedriver
fi

# The env setup is intentionally separate into a separate file
# so that we can safely source it from scripts that could be
# sourced from smartcd and the like

source "${BASH_SOURCE%/*}/commonEnv.bash"


# TODO switch to using gg

gg="go generate"


# *****************************************
LOADED_COMMON_BASH="true"
# *****************************************

source "${BASH_SOURCE%/*}/setupPATH.bash"
