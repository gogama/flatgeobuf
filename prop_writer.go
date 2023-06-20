// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"io"
	"math"
	"unsafe"

	flatbuffers "github.com/google/flatbuffers/go"
)

// PropWriter writes a list of key value pairs in FlatGeobuf property
// format to an underlying stream.
type PropWriter struct {
	w io.Writer
}

// TODO: Docs
func NewPropWriter(w io.Writer) *PropWriter {
	if w == nil {
		textPanic("nil writer")
	}
	return &PropWriter{w: w}
}

// TODO: Docs
func (w *PropWriter) WriteByte(v int8) (n int, err error) {
	b := []byte{byte(v)}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteUByte(v uint8) (n int, err error) {
	b := []byte{v}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteBool(v bool) (n int, err error) {
	b := []byte{0}
	if v {
		b[0] = 1
	}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteShort(v int16) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8)}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteUShort(v uint16) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8)}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteInt(v int32) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteUInt(v uint32) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteLong(v int64) (n int, err error) {
	b := []byte{
		byte(v >> 000), byte(v >> 010), byte(v >> 020), byte(v >> 030),
		byte(v >> 040), byte(v >> 050), byte(v >> 060), byte(v >> 070),
	}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteULong(v uint64) (n int, err error) {
	b := []byte{
		byte(v >> 000), byte(v >> 010), byte(v >> 020), byte(v >> 030),
		byte(v >> 040), byte(v >> 050), byte(v >> 060), byte(v >> 070),
	}
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteFloat(v float32) (n int, err error) {
	b := make([]byte, flatbuffers.SizeFloat32)
	flatbuffers.WriteFloat32(b, v)
	return w.w.Write(b)
}

// TODO: Docs
func (w *PropWriter) WriteDouble(v float64) (n int, err error) {
	b := make([]byte, flatbuffers.SizeFloat64)
	flatbuffers.WriteFloat64(b, v)
	return w.w.Write(b)
}

// TODO: Docs, they should also use for String
func (w *PropWriter) WriteString(v string) (n int, err error) {
	return w.WriteBinary(unsafe.Slice(unsafe.StringData(v), len(v)))
}

// TOdO: Docs, they should also use for JSON
func (w *PropWriter) WriteBinary(v []byte) (n int, err error) {
	if int64(len(v)) > math.MaxUint32 {
		return 0, fmtErr("property length %d overflows uint32", len(v))
	}
	n, err = w.WriteUInt(uint32(len(v)))
	if err != nil {
		return
	}
	var m int
	m, err = w.w.Write(v)
	n += m
	return
}
