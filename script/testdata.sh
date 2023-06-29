#!/usr/bin/env bash

# Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
# Use of this source code is governed by an MIT-style
# license that can be found in the LICENSE file.

set -eo pipefail

if ! command -v rsync >/dev/null; then
  >&2 << EOF cat
error: rsync not found
  | This script uses rsync to synchronize test data files from
  | https://github.com/flatgeobuf/flatgeobuf into the directory
  | testdata/.
  |
  | Install rsync if you want to proceed.
EOF
  exit 2
fi

if ! [ -d tmp ]; then
  mkdir tmp
fi

if ! [ -d tmp/flatgeobuf ]; then
  git clone git@github.com:flatgeobuf/flatgeobuf.git tmp/flatgeobuf
else
  pushd tmp/flatgeobuf
  git pull
  popd
fi

rsync -av --delete \
  --include "*/" \
  --include "*.fgb" \
  --exclude "*" \
  tmp/flatgeobuf/test/data/ ../testdata/flatgeobuf/
