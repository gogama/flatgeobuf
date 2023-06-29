// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree

import (
	"bytes"
	"sort"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

var octets = []uint64{
	uint64(0x0706050403020100),
	uint64(0x0f0e0d0c0b0a0908),
	uint64(0x1716151413121110),
	uint64(0x1f1e1d1c1b1a1918),
}

func octetBytes() []byte {
	dup := make([]uint64, len(octets))
	copy(dup, octets)
	ptr := (*byte)(unsafe.Pointer(&dup[0]))
	return unsafe.Slice(ptr, 8*len(dup))
}

func TestFixLittleEndianOctets(t *testing.T) {
	b := octetBytes()

	fixLittleEndianOctets(b)

	assert.True(t, sort.SliceIsSorted(b, func(i, j int) bool {
		return b[i] < b[j]
	}))
}

func TestWriteLittleEndianOctets(t *testing.T) {
	var b bytes.Buffer

	n, err := writeLittleEndianOctets(&b, octetBytes())

	assert.NoError(t, err)
	assert.Equal(t, n, 8*len(octets))
	assert.True(t, sort.SliceIsSorted(b.Bytes(), func(i, j int) bool {
		return b.Bytes()[i] < b.Bytes()[j]
	}))
}
