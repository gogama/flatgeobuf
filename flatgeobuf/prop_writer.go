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

// PropWriter writes a list of key/value pairs in FlatGeobuf property
// format to an underlying stream.
//
// Each FlatGeobuf feature table (flat.Feature) contains an optional
// byte  array field named Properties which is encoded in its own custom
// format, a format-within-a-format, if you will. PropWriter knows how
// to write this special format-within-a-format.
//
// A typical usage pattern is to write the properties for a feature to
// a byte buffer using a PropWriter, convert the buffer containing the
// properties into a FlatBuffer byte vector (with flatbuffers.Builder),
// and finally supply the vector offset when building the feature using
// flat.FeatureAddProperties.
//
// Use WriteString for flat.ColumnTypeString and flat.ColumnTypeDateTime.
// Use WriteBinary for flat.ColumnTypeBinary and flat.ColumnTypeJson.
type PropWriter struct {
	w io.Writer
}

// NewPropWriter creates a new FlatGeobuf feature property writer based
// on an underlying output stream.
func NewPropWriter(w io.Writer) *PropWriter {
	if w == nil {
		textPanic("nil writer")
	}
	return &PropWriter{w: w}
}

// WriteByte writes the value of a flat.ColumnTypeByte property (signed
// byte value).
func (w *PropWriter) WriteByte(v int8) (n int, err error) {
	b := []byte{byte(v)}
	return w.w.Write(b)
}

// WriteUByte writes the value of a flat.ColumnTypeUByte property
// (unsigned byte value).
func (w *PropWriter) WriteUByte(v uint8) (n int, err error) {
	b := []byte{v}
	return w.w.Write(b)
}

// WriteBool writes the value of a flat.ColumnTypeBool property (a byte
// value of zero for false, one for true).
func (w *PropWriter) WriteBool(v bool) (n int, err error) {
	b := []byte{0}
	if v {
		b[0] = 1
	}
	return w.w.Write(b)
}

// WriteShort writes the value of a flat.ColumnTypeShort property (a
// 16-bit signed integer value).
func (w *PropWriter) WriteShort(v int16) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8)}
	return w.w.Write(b)
}

// WriteUShort writes the value of a flat.ColumnTypeUShort property (a
// 16-bit unsigned integer value).
func (w *PropWriter) WriteUShort(v uint16) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8)}
	return w.w.Write(b)
}

// WriteInt writes the value of a flat.ColumnTypeInt property (a 32-bit
// signed integer value).
func (w *PropWriter) WriteInt(v int32) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
	return w.w.Write(b)
}

// WriteUInt writes the value of a flat.ColumnTypeUInt property (a
// 32-bit unsigned integer value).
func (w *PropWriter) WriteUInt(v uint32) (n int, err error) {
	b := []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
	return w.w.Write(b)
}

// WriteLong writes the value of a flat.ColumnTypeLong property (a
// 64-bit signed integer value).
func (w *PropWriter) WriteLong(v int64) (n int, err error) {
	b := []byte{
		byte(v >> 000), byte(v >> 010), byte(v >> 020), byte(v >> 030),
		byte(v >> 040), byte(v >> 050), byte(v >> 060), byte(v >> 070),
	}
	return w.w.Write(b)
}

// WriteULong writes the value of a flat.ColumnTypeULong property (a
// 64-bit unsigned integer value).
func (w *PropWriter) WriteULong(v uint64) (n int, err error) {
	b := []byte{
		byte(v >> 000), byte(v >> 010), byte(v >> 020), byte(v >> 030),
		byte(v >> 040), byte(v >> 050), byte(v >> 060), byte(v >> 070),
	}
	return w.w.Write(b)
}

// WriteFloat writes the value of a flat.ColumnTypeFloat property (an
// IEEE 32-bit single precision floating point number).
func (w *PropWriter) WriteFloat(v float32) (n int, err error) {
	b := make([]byte, flatbuffers.SizeFloat32)
	flatbuffers.WriteFloat32(b, v)
	return w.w.Write(b)
}

// WriteDouble writes the value of a flat.ColumnTypeDouble property (an
// IEEE 64-bit double precision floating point number).
func (w *PropWriter) WriteDouble(v float64) (n int, err error) {
	b := make([]byte, flatbuffers.SizeFloat64)
	flatbuffers.WriteFloat64(b, v)
	return w.w.Write(b)
}

// WriteString writes the value of a string property of type
// flat.ColumnTypeString.
//
// WriteString can also be used to write a value of type
// flat.ColumnTypeDateTime provided the value is serialized to a string
// in the ISO-8601 format.
func (w *PropWriter) WriteString(v string) (n int, err error) {
	return w.WriteBinary(unsafe.Slice(unsafe.StringData(v), len(v)))
}

// WriteBinary writes an arbitrary-length property value, which can be
// either flat.ColumnTypeBinary or a flat.ColumnTypeJson.
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
