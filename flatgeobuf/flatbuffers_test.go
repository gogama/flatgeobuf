// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"errors"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

func Test_safeFlatBuffersInteraction(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Function Panicked", func(t *testing.T) {
			err := safeFlatBuffersInteraction(func() error { panic("oh noes") })

			assert.EqualError(t, err, "panic: flatbuffers: oh noes")
		})

		t.Run("Function Errored", func(t *testing.T) {
			expectedErr := errors.New("very problematic statements")

			actualErr := safeFlatBuffersInteraction(func() error { return expectedErr })

			assert.Same(t, expectedErr, actualErr)
		})
	})

	t.Run("Success", func(t *testing.T) {
		didAnythingHappen := false

		err := safeFlatBuffersInteraction(func() error {
			didAnythingHappen = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, didAnythingHappen)
	})
}

func Test_writeSizePrefixedTable(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Table Size Too Small for Size Prefix", func(t *testing.T) {
			w := newMockWriteCloser(t)

			n, err := writeSizePrefixedTable(w, flatbuffers.Table{})

			assert.EqualError(t, err, "flatgeobuf: FlatBuffers table buffer is too small for a size prefix (need=4, len=0)")
			assert.Equal(t, 0, n)
			w.AssertExpectations(t)
		})

		t.Run("Table Buffer is Smaller Than Size Prefix Indicates", func(t *testing.T) {
			w := newMockWriteCloser(t)

			n, err := writeSizePrefixedTable(w, flatbuffers.Table{
				Bytes: []byte{0x02, 0x00, 0x00, 0x00, 0xff},
			})

			assert.EqualError(t, err, "flatgeobuf: FlatBuffers table buffer is smaller than size prefix indicates (need=4+2, len=5, gap=1)")
			assert.Equal(t, 0, n)
			w.AssertExpectations(t)
		})

		t.Run("Write Error", func(t *testing.T) {
			expectedErr := errors.New("encountering choppy conditions")
			expectedBytes := make([]byte, flatbuffers.SizeUint32)
			dup := make([]byte, flatbuffers.SizeUint32)
			w := newMockWriteCloser(t)
			w.
				On("Write", expectedBytes).
				Return(3, expectedErr).
				Once()

			n, err := writeSizePrefixedTable(w, flatbuffers.Table{Bytes: dup})

			assert.Same(t, err, expectedErr)
			assert.Equal(t, 3, n)
			w.AssertExpectations(t)
		})
	})

	t.Run("Success", func(t *testing.T) {
		const numDataBytes = 256
		expectedBytes := make([]byte, flatbuffers.SizeUint32+numDataBytes)
		flatbuffers.WriteUint32(expectedBytes, uint32(numDataBytes))
		for i := 0; i < numDataBytes; i++ {
			expectedBytes[flatbuffers.SizeUint32+i] = byte(i)
		}
		dup := make([]byte, len(expectedBytes))
		copy(dup, expectedBytes)
		w := newMockWriteCloser(t)
		w.
			On("Write", expectedBytes).
			Return(len(expectedBytes), nil).
			Once()

		n, err := writeSizePrefixedTable(w, flatbuffers.Table{Bytes: dup})

		assert.NoError(t, err)
		assert.Equal(t, len(expectedBytes), n)
		w.AssertExpectations(t)
	})
}

func Test_tableSize(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Too Small", func(t *testing.T) {
			size, err := tableSize(flatbuffers.Table{})

			assert.EqualError(t, err, "flatgeobuf: FlatBuffers table buffer is too small for a size prefix (need=4, len=0)")
			assert.Equal(t, uint32(0), size)
		})
	})

	t.Run("Success", func(t *testing.T) {
		const n = 1234567
		tbl := flatbuffers.Table{
			Bytes: make([]byte, flatbuffers.SizeUint32),
		}
		flatbuffers.WriteUint32(tbl.Bytes, n)

		size, err := tableSize(tbl)

		assert.NoError(t, err)
		assert.Equal(t, uint32(n), size)
	})
}
