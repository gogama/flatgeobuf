// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"fmt"
	"io"

	flatbuffers "github.com/google/flatbuffers/go"
)

// safeFlatBuffersInteraction runs a function that interacts with
// FlatBuffers, trapping any panic that occurs and converting it to a
// normal Go error.
//
// This function exists because FlatBuffer's Go code doesn't use
// standard Go error handling, allegedly for performance reasons, and
// consequently any invalid attempt to interact with FlatBuffer data
// may trigger a panic.
func safeFlatBuffersInteraction(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: flatbuffers: %v", r)
		}
	}()
	err = f()
	return
}

// writeSizePrefixedTable writes a size-prefixed root FlatBuffers table
// which is positioned at offset zero of its buffer to an output stream.
// We have to put the size-prefixed, root, and offset zero constraints
// on the input table, because otherwise it is impossible to know the
// table's size or ensure that it occupies contiguous bytes.
func writeSizePrefixedTable(w io.Writer, t flatbuffers.Table) (n int, err error) {
	var size uint32
	if size, err = tableSize(t); err != nil {
		return
	} else if uint64(size) > uint64(len(t.Bytes)) {
		err = fmtErr("FlatBuffers table buffer is smaller than the size prefix (Len=%d, size=%d)", len(t.Bytes), size)
		return
	} else {
		n, err = w.Write(t.Bytes[0 : flatbuffers.SizeUint32+size])
		return
	}
}

func tableSize(t flatbuffers.Table) (size uint32, err error) {
	if t.Pos != flatbuffers.SizeUint32 {
		err = fmtErr("not a size-prefixed root FlatBuffers table at offset 0 (Len=%d, Pos=%d)", len(t.Bytes), t.Pos)
	}
	size = flatbuffers.GetUint32(t.Bytes)
	return
}
