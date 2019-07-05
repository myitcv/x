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

source "${BASH_SOURCE%/*}/commonFunctions.bash"

# The env setup is intentionally separate into a separate file
# so that we can safely source it from scripts that could be
# sourced from smartcd and the like

source "${BASH_SOURCE%/*}/commonEnv.bash"

# Here we "override" the version of Go if USE_GO_TIP is set.
# This is needed where a Go version is not available via Travis

if [ $(running_on_ci_server) == "yes" ]
then
	export PATH="$HOME/go/bin:$PATH"
fi

# TODO switch to using gogenerate
gg="go generate"

# *****************************************
autostash_or_export LOADED_COMMON_BASH="true"
# *****************************************

source "${BASH_SOURCE%/*}/setupPATH.bash"
