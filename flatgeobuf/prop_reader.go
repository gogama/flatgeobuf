// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"unsafe"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	flatbuffers "github.com/google/flatbuffers/go"
)

// PropReader reads a list of key/value pairs in FlatGeobuf property
// format from an underlying stream.
//
// Each FlatGeobuf feature table (flat.Feature) contains an optional
// byte array field named Properties which is encoded in its own custom
// format, a format-within-a-format, if you will. PropReader knows how
// to read this special format-within-a-format.
//
// To read all properties at once for a given feature property Schema,
// use ReadSchema.
//
// Use ReadString for flat.ColumnTypeString and flat.ColumnTypeDateTime.
// Use ReadBinary for flat.ColumnTypeBinary and flat.ColumnTypeJson.
type PropReader struct {
	// r is the stream to read from.
	r io.Reader
}

// NewPropReader creates a new FlatGeobuf feature property reader
// reading from an underlying input stream.
func NewPropReader(r io.Reader) *PropReader {
	if r == nil {
		textPanic("nil reader")
	}
	return &PropReader{r: r}
}

// ReadByte reads the value of a flat.ColumnTypeByte property (signed
// byte value).
func (r *PropReader) ReadByte() (int8, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return int8(b[0]), nil
}

// ReadUByte reads the value of a flat.ColumnTypeUByte property
// (unsigned byte value).
func (r *PropReader) ReadUByte() (uint8, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

// ReadBool reads the value of a flat.ColumnTypeBool property (a byte
// value of zero for false, one for true).
func (r *PropReader) ReadBool() (bool, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return false, err
	}
	return b[0] > 0, nil
}

// ReadShort reads the value of a flat.ColumnTypeShort property (a
// 16-bit signed integer value).
func (r *PropReader) ReadShort() (int16, error) {
	b := make([]byte, 2)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return int16(b[0]) | int16(b[1])<<8, nil
}

// ReadUShort reads the value of a flat.ColumnTypeUShort property (a
// 16-bit unsigned integer value).
func (r *PropReader) ReadUShort() (uint16, error) {
	b := make([]byte, 2)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return uint16(b[0]) | uint16(b[1])<<8, nil
}

// ReadInt writes the value of a flat.ColumnTypeInt property (a 32-bit
// signed integer value).
func (r *PropReader) ReadInt() (int32, error) {
	b := make([]byte, 4)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16 | int32(b[3])<<24, nil
}

// ReadUInt reads the value of a flat.ColumnTypeUInt property (a 32-bit
// unsigned integer value).
func (r *PropReader) ReadUInt() (uint32, error) {
	b := make([]byte, 4)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24, nil
}

// ReadLong reads the value of a flat.ColumnTypeLong property (a 64-bit
// signed integer value).
func (r *PropReader) ReadLong() (int64, error) {
	b := make([]byte, 8)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	v := int64(b[0])<<000 | int64(b[1])<<010 | int64(b[2])<<020 | int64(b[3])<<030 |
		int64(b[4])<<040 | int64(b[5])<<050 | int64(b[6])<<060 | int64(b[7])<<070
	return v, nil
}

// ReadULong reads the value of a flat.ColumnTypeULong property (a
// 64-bit unsigned integer value).
func (r *PropReader) ReadULong() (uint64, error) {
	b := make([]byte, 8)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	v := uint64(b[0])<<000 | uint64(b[1])<<010 | uint64(b[2])<<020 | uint64(b[3])<<030 |
		uint64(b[4])<<040 | uint64(b[5])<<050 | uint64(b[6])<<060 | uint64(b[7])<<070
	return v, nil
}

// ReadFloat reads the value of a flat.ColumnTypeFloat property (an
// IEEE 32-bit single precision floating point number).
func (r *PropReader) ReadFloat() (float32, error) {
	b := make([]byte, flatbuffers.SizeFloat32)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return flatbuffers.GetFloat32(b), nil
}

// ReadDouble reads the value of a flat.ColumnTypeDouble property (an
// IEEE 64-bit double precision floating point number).
func (r *PropReader) ReadDouble() (float64, error) {
	b := make([]byte, flatbuffers.SizeFloat64)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return flatbuffers.GetFloat64(b), nil
}

// ReadString reads the value of a string property of type
// flat.ColumnTypeString.
//
// ReadString can also be used to read a value of type
// flat.ColumnTypeDateTime, since FlatGeobuf encodes date/time values as
// strings serialized in the ISO-8601 format.
func (r *PropReader) ReadString() (string, error) {
	b, err := r.ReadBinary()
	if err != nil {
		return "", err
	}
	if len(b) == 0 {
		return "", nil
	}
	return unsafe.String(&b[0], len(b)), nil
}

// ReadBinary reads an arbitrary-length property value, which can be
// either flat.ColumnTypeBinary or a flat.ColumnTypeJson.
func (r *PropReader) ReadBinary() ([]byte, error) {
	n, err := r.ReadUInt()
	if err != nil {
		return nil, err
	}
	if int64(n) > math.MaxInt {
		return nil, fmtErr("property length %d overflows int", n)
	}
	b := make([]byte, int(n))
	if _, err = io.ReadFull(r.r, b); err != nil {
		return nil, err
	}
	return b, nil
}

// PropValue pairs together a FlatGeobuf feature property "key" (a
// flat.Column reference) with the property value and the type of its
// value.
type PropValue struct {
	// Col is the FlatGeobuf column table describing the property's
	// column, or key.
	Col flat.Column
	// Value is the deserialized property value.
	Value any
	// ColIndex is the zero-based index of the property within the
	// feature's column schema.
	ColIndex uint16
	// Type is the FlatGeobuf column type. This value is repeated in the
	// Type field for ease of consumption, and is also available through
	// Col.
	Type flat.ColumnType
}

func (val PropValue) String() string {
	var name []byte
	_ = safeFlatBuffersInteraction(func() error {
		name = val.Col.Name()
		return nil
	})

	var b strings.Builder
	if name != nil {
		b.WriteString("PropValue{Name:\"")
		b.Write(name)
		b.WriteString("\",")
	} else {
		b.WriteString("PropValue{")
	}

	_, _ = fmt.Fprintf(&b, "Type:%s,Value:", val.Type)

	switch val.Type {
	case flat.ColumnTypeUByte, flat.ColumnTypeUShort, flat.ColumnTypeUInt, flat.ColumnTypeULong:
		_, _ = fmt.Fprintf(&b, "0x%x", val.Value)
	case flat.ColumnTypeBool:
		_, _ = fmt.Fprintf(&b, "%t", val.Value)
	case flat.ColumnTypeString:
		_, _ = fmt.Fprintf(&b, "%q", val.Value)
	default:
		_, _ = fmt.Fprintf(&b, "%v", val.Value)
	}

	_, _ = fmt.Fprintf(&b, ",ColIndex:%d}", val.ColIndex)

	return b.String()
}

// ReadSchema all properties specified in the given Schema, returning
// them as a slice of PropValue structures.
//
// The concrete implementation of the schema will typically be a
// *flat.Header or a *flat.Feature.
func (r *PropReader) ReadSchema(schema Schema) ([]PropValue, error) {
	var n int
	if err := safeFlatBuffersInteraction(func() error {
		n = schema.ColumnsLength()
		return nil
	}); err != nil {
		return nil, wrapErr("failed to read schema column count", err)
	}
	vals := make([]PropValue, 0, n)

	for i := 0; i < n; i++ {
		col, err := r.ReadUShort()
		if err != nil {
			return nil, wrapErr("failed to read column index (for property %d of %d)", err, i, n)
		}
		j := int(col)
		if j >= n {
			return nil, fmtErr("schema has only %d columns, but property %d has column index %d", n, i, j)
		}
		val := PropValue{
			ColIndex: col,
		}
		if err = safeFlatBuffersInteraction(func() error {
			if !schema.Columns(&val.Col, j) {
				return errors.New("column not found")
			}
			return nil
		}); err != nil {
			return nil, wrapErr("failed to fetch column %d (for property %d of %d)", err, j, i, n)
		}
		val.Type = val.Col.Type()
		switch val.Type {
		case flat.ColumnTypeByte:
			val.Value, err = r.ReadByte()
		case flat.ColumnTypeUByte:
			val.Value, err = r.ReadUByte()
		case flat.ColumnTypeBool:
			val.Value, err = r.ReadBool()
		case flat.ColumnTypeShort:
			val.Value, err = r.ReadShort()
		case flat.ColumnTypeUShort:
			val.Value, err = r.ReadUShort()
		case flat.ColumnTypeInt:
			val.Value, err = r.ReadInt()
		case flat.ColumnTypeUInt:
			val.Value, err = r.ReadUInt()
		case flat.ColumnTypeLong:
			val.Value, err = r.ReadLong()
		case flat.ColumnTypeULong:
			val.Value, err = r.ReadULong()
		case flat.ColumnTypeFloat:
			val.Value, err = r.ReadFloat()
		case flat.ColumnTypeDouble:
			val.Value, err = r.ReadDouble()
		case flat.ColumnTypeString, flat.ColumnTypeDateTime:
			val.Value, err = r.ReadString()
		case flat.ColumnTypeJson, flat.ColumnTypeBinary:
			val.Value, err = r.ReadBinary()
		default:
			fmtPanic("unknown column type: %s", val.Type)
		}
		vals = append(vals, val)
	}

	return vals, nil
}
