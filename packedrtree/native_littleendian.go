// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

//go:build 386 || amd64 || amd64p32 || arm || arm64 || loong64 || mipsle || mis64le || mips64p32le || ppc64le || riscv || riscv64 || wasm

package packedrtree

import "io"

func fixLittleEndianOctets(_ []byte) {} // No-op since architecture is little-endian.

func writeLittleEndianOctets(w io.Writer, p []byte) (int, error) {
	return w.Write(p)
}
