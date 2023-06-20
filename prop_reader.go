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

// PropReader reads a list of key value pairs in FlatGeobuf property
// format from an underlying stream.
//
// FIXME: NEED TO DOCUMENT UNSAFE ASPECT -- They aren't allowed to mod
//
//	the buffer after.
type PropReader struct {
	// r is the stream to read from.
	r io.Reader
}

func NewPropReader(r io.Reader) *PropReader {
	if r == nil {
		textPanic("nil reader")
	}
	return &PropReader{r: r}
}

func (r *PropReader) ReadByte() (int8, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return int8(b[0]), nil
}

func (r *PropReader) ReadUByte() (uint8, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *PropReader) ReadBool() (bool, error) {
	b := make([]byte, 1)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return false, err
	}
	return b[0] > 0, nil
}

func (r *PropReader) ReadShort() (int16, error) {
	b := make([]byte, 2)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return int16(b[0]) | int16(b[1]<<8), nil
}

func (r *PropReader) ReadUShort() (uint16, error) {
	b := make([]byte, 2)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return uint16(b[0]) | uint16(b[1]<<8), nil
}

func (r *PropReader) ReadInt() (int32, error) {
	b := make([]byte, 4)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return int32(b[0]) | int32(b[1]<<8) | int32(b[2]<<16) | int32(b[3]<<24), nil
}

func (r *PropReader) ReadUInt() (uint32, error) {
	b := make([]byte, 4)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return uint32(b[0]) | uint32(b[1]<<8) | uint32(b[2]<<16) | uint32(b[3]<<24), nil
}

func (r *PropReader) ReadLong() (int64, error) {
	b := make([]byte, 8)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	v := int64(b[0]<<000) | int64(b[1]<<010) | int64(b[2]<<020) | int64(b[3]<<030) |
		int64(b[4]<<040) | int64(b[5]<<050) | int64(b[6]<<060) | int64(b[7]<<070)
	return v, nil
}

func (r *PropReader) ReadULong() (uint64, error) {
	b := make([]byte, 8)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	v := uint64(b[0]<<000) | uint64(b[1]<<010) | uint64(b[2]<<020) | uint64(b[3]<<030) |
		uint64(b[4]<<040) | uint64(b[5]<<050) | uint64(b[6]<<060) | uint64(b[7]<<070)
	return v, nil
}

func (r *PropReader) ReadFloat() (float32, error) {
	b := make([]byte, flatbuffers.SizeFloat32)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return flatbuffers.GetFloat32(b), nil
}

func (r *PropReader) ReadDouble() (float64, error) {
	b := make([]byte, flatbuffers.SizeFloat64)
	_, err := io.ReadFull(r.r, b)
	if err != nil {
		return 0, err
	}
	return flatbuffers.GetFloat64(b), nil
}

// TODO: Docs, they should also use for DateTime
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

type PropValue struct {
	Col      Column
	Value    interface{}
	ColIndex uint16
	Type     ColumnType
}

func (r *PropReader) ReadSchema(schema Schema) ([]PropValue, error) {
	n := schema.ColumnsLength()
	vals := make([]PropValue, 0, n)

	for {
		col, err := r.ReadUShort()
		if err == io.EOF {
			return vals, nil
		} else if err != nil {
			return nil, fmtErr("error reading column index")
		}
		i := int(col)
		if i >= n {
			return nil, fmtErr("column index %d not in schema (%d columns)", i, n)
		}
		val := PropValue{
			ColIndex: col,
		}
		if !schema.Columns(&val.Col, i) {
			return nil, fmtErr("schema failed to locate column %d", i)
		}
		val.Type = val.Col.Type()
		switch val.Type {
		case ColumnTypeByte:
			val.Value, err = r.ReadByte()
		case ColumnTypeUByte:
			val.Value, err = r.ReadUByte()
		case ColumnTypeBool:
			val.Value, err = r.ReadBool()
		case ColumnTypeShort:
			val.Value, err = r.ReadShort()
		case ColumnTypeUShort:
			val.Value, err = r.ReadUShort()
		case ColumnTypeInt:
			val.Value, err = r.ReadInt()
		case ColumnTypeUInt:
			val.Value, err = r.ReadUInt()
		case ColumnTypeLong:
			val.Value, err = r.ReadLong()
		case ColumnTypeULong:
			val.Value, err = r.ReadULong()
		case ColumnTypeFloat:
			val.Value, err = r.ReadFloat()
		case ColumnTypeDouble:
			val.Value, err = r.ReadDouble()
		case ColumnTypeString, ColumnTypeDateTime:
			val.Value, err = r.ReadString()
		case ColumnTypeJson, ColumnTypeBinary:
			val.Value, err = r.ReadBinary()
		default:
			fmtPanic("unknown column type: %s", val.Type)
		}
		vals = append(vals, val)
	}
}
