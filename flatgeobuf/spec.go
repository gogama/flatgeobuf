// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"io"
)

const (
	// magicLen is the length of the FlatGeobuf magic number in bytes.
	magicLen = 8
	// MinSpecMajorVersion is the minimum major version of the
	// FlatGeobuf specification that this package can read.
	MinSpecMajorVersion = 0x03
	// MaxSpecMajorVersion is the maximum major version of the
	// FlatGeobuf specification that this package can read.
	MaxSpecMajorVersion = 0x03
	// headerMaxLen is an artificial limit, not imposed by the
	// FlatGeobuf specification, on the maximum size of a FlatGeobuf
	// file header this package will read. The purpose of this value
	// is to impose some limitation, to prevent corrupted or malicious
	// file headers from causing huge and pointless memory allocations.
	headerMaxLen = 32 * 1024 * 1024
)

// magic contains the FlatGeobuf magic number.
//
// The fourth byte is the specification major version of data written
// by this package, and the last byte is the specification patch
// version of data written by this package.
var magic = [magicLen]byte{0x66, 0x67, 0x62, 0x03, 0x66, 0x67, 0x62, 0x01}

// SpecVersion is a version of the FlatGeobuf specification.
type SpecVersion struct {
	// Major is the major version of the FlatGeobuf specification.
	Major uint8
	// Patch is the patch version of the FlatGeobuf specification.
	Patch uint8
}

// Magic reads the FlatGeobuf magic number from a stream and if it is
// valid, returns the FlatGeobuf specification version. This function
// can be used to test whether any file seems to be in the FlatGeobuf
// format. However, it does not read beyond the magic number.
//
// Calling this function will result in 8 bytes being read from the
// stream reader (unless there were fewer than 8 bytes available, in
// which all available bytes in the stream are consumed).
func Magic(r io.Reader) (SpecVersion, error) {
	m := make([]byte, magicLen)
	_, err := io.ReadFull(r, m)
	if err != nil {
		return SpecVersion{}, err
	}
	if m[0] == magic[0] &&
		m[1] == magic[1] &&
		m[2] == magic[2] &&
		m[4] == magic[4] &&
		m[5] == magic[5] &&
		m[6] == magic[6] {
		return SpecVersion{m[3], m[7]}, nil
	}
	return SpecVersion{}, textErr("invalid magic number")
}
