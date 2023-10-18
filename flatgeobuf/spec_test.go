// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMagic_ErrRead(t *testing.T) {
	r := bytes.NewReader([]byte{magic[0]})

	version, err := Magic(r)

	assert.Zero(t, version)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestMagic_ErrInvalid(t *testing.T) {
	b := make([]byte, len(magic))
	copy(b, magic[:])
	b[len(magic)-2] += 1
	r := bytes.NewReader(b)

	version, err := Magic(r)

	assert.Zero(t, version)
	assert.EqualError(t, err, "flatgeobuf: invalid magic number")
}

func TestMagic_Success(t *testing.T) {
	b := make([]byte, len(magic))
	copy(b, magic[:])
	r := bytes.NewReader(b)

	version, err := Magic(r)

	assert.Equal(t, SpecVersion{magic[3], magic[7]}, version)
	assert.NoError(t, err)
}
