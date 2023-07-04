#!/usr/bin/env bash

# Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
# Use of this source code is governed by an MIT-style
# license that can be found in the LICENSE file.

set -eo pipefail

if ! command -v flatc >/dev/null; then
  >&2 << EOF cat
error: Flatbuffers compiler not found: flatc
  | Find a pre-built binary for your OS here:
  |   https://github.com/google/flatbuffers/releases/
  |
  | CMake hell is probably not worth it for you, but if it is, by all
  | means feel free to try:
  |   https://flatbuffers.dev/flatbuffers_guide_building.html.
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

# There doesn't seem to be a way to stop flatc from writing the
# output files into a directory with the same name as the go namespace.
# Hence using `-o ../flatgeobuf/` with namespace `flat` results in flatc
# generating the files under `../flatgeobuf/flat/`.
flatc --go -o ../flatgeobuf/ --go-namespace flat tmp/flatgeobuf/src/fbs/*.fbs

(cd tmp/flatgeobuf && git describe --tags) >../flatgeobuf/flat/version-schema.txt
flatc --version >../flatgeobuf/flat/version-flatc.txt
