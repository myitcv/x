#!/usr/bin/env vbash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

source "$(git rev-parse --show-toplevel)/_scripts/common.bash"

./_scripts/install_go.sh
./_scripts/setupCIEnv.sh
./_scripts/run_tests.sh
./_scripts/check_code_is_formatted.sh
./_scripts/check_git_is_clean.sh
