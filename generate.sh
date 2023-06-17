#!/usr/bin/env bash

# Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
# Use of this source code is governed by an MIT-style
# license that can be found in the LICENSE file.


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

if ! [ -d flatgeobuf ]; then
  git clone git@github.com:flatgeobuf/flatgeobuf.git
fi

# There doesn't seem to be a way to stop flatc from writing the
# output files into a directory with the same name as the go namespace.
# Hence using `-o ..` is a clever trick, based on the fact that this
# repo's directory should be named flatgeobuf, to get the files
# generated right in the root.
flatc --go -o .. --go-namespace flatgeobuf flatgeobuf/src/fbs/*.fbs

(cd flatgeobuf && git describe --tags) >version-schema.txt
flatc --version >version-flatc.txt
