#!/usr/bin/env vbash

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

only_run_on_ci_server

# Ensure cache directories etc exist

mkdir -p $CI_CACHE_DIR
mkdir -p $CI_DEPENDENCIES_DIR
mkdir -p $PROTOBUF_INSTALL_DIR

# Install protobuf and other external deps

"${BASH_SOURCE%/*}/install_protobuf.sh"
"${BASH_SOURCE%/*}/install_chromedriver.sh"
