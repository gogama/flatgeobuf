// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testPropValue struct {
	ser []byte
	de  any
}

var testPropValues = map[flat.ColumnType][]testPropValue{
	flat.ColumnTypeByte: {
		{[]byte{0x9c}, int8(-100)},
		{[]byte{0x00}, int8(0)},
		{[]byte{0x7f}, int8(127)},
	},
	flat.ColumnTypeUByte: {
		{[]byte{0x00}, uint8(0)},
		{[]byte{0x01}, uint8(1)},
		{[]byte{0x7f}, uint8(127)},
		{[]byte{0xff}, uint8(255)},
	},
	flat.ColumnTypeBool: {
		{[]byte{0x00}, false},
		{[]byte{0x01}, true},
	},
	flat.ColumnTypeShort: {
		{[]byte{0x01, 0x80}, int16(-32767)},
		{[]byte{0x00, 0x00}, int16(0)},
		{[]byte{0x10, 0x01}, int16(272)},
	},
	flat.ColumnTypeUShort: {
		{[]byte{0x00, 0x00}, uint16(0)},
		{[]byte{0x10, 0x01}, uint16(272)},
		{[]byte{0x01, 0x80}, uint16(32769)},
	},
	flat.ColumnTypeInt: {
		{[]byte{0x9a, 0x0b, 0xc0, 0x80}, int32(-2_134_897_766)},
		{[]byte{0x00, 0x00, 0x00, 0x00}, int32(0)},
		{[]byte{0x01, 0x11, 0x22, 0x7f}, int32(2_132_939_009)},
	},
	flat.ColumnTypeUInt: {
		{[]byte{0x00, 0x00, 0x00, 0x00}, uint32(0)},
		{[]byte{0x04, 0x03, 0x02, 0x01}, uint32(0x01020304)},
		{[]byte{0xc0, 0xd0, 0xe0, 0xf0}, uint32(0xf0e0d0c0)},
		{[]byte{0xef, 0xbe, 0xad, 0xde}, uint32(0xdeadbeef)},
	},
	flat.ColumnTypeLong: {
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80}, int64(-9_223_372_036_854_775_808)},
		{[]byte{0x10, 0x32, 0x54, 0x76, 0x98, 0xba, 0xdc, 0xfe}, int64(-81_985_529_216_486_896)},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, int64(0)},
		{[]byte{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01}, int64(81_985_529_216_486_895)},
	},
	flat.ColumnTypeULong: {
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, uint64(0)},
		{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, uint64(0x0706050403020100)},
		{[]byte{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88}, uint64(0x8899aabbccddeeff)},
	},
	flat.ColumnTypeFloat: {
		{[]byte{0x00, 0x00, 0xc0, 0x7f}, float32(math.NaN())},
		{[]byte{0x00, 0x00, 0x80, 0xff}, float32(math.Inf(-1))},
		{[]byte{0x80, 0x96, 0x18, 0xcb}, float32(-1e7)},
		{[]byte{0x00, 0x00, 0x00, 0x00}, float32(0.0)},
		{[]byte{0x80, 0x96, 0x18, 0x4b}, float32(1e7)},
		{[]byte{0x00, 0x00, 0x80, 0x7f}, float32(math.Inf(1))},
	},
	flat.ColumnTypeDouble: {
		{[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf8, 0x7f}, math.NaN()},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0xff}, math.Inf(-1)},
		{[]byte{0x40, 0x8c, 0xb5, 0x78, 0x1d, 0xaf, 0x15, 0xc4}, -1e20},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 0.0},
		{[]byte{0x40, 0x8c, 0xb5, 0x78, 0x1d, 0xaf, 0x15, 0x44}, 1e20},
		{[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf0, 0x7f}, math.Inf(1)},
	},
	flat.ColumnTypeString: {
		{[]byte{0x00, 0x00, 0x00, 0x00}, ""},
		{[]byte{0x01, 0x00, 0x00, 0x00, 0x20}, " "},
		{[]byte{0x04, 0x00, 0x00, 0x00, 0x46, 0x30, 0x4f, 0x21}, "F0O!"},
		{[]byte{0x18, 0x00, 0x00, 0x00, 0x32, 0x30, 0x32, 0x33, 0x2d, 0x31, 0x30, 0x2d, 0x31, 0x37, 0x54, 0x30, 0x30, 0x3a, 0x31, 0x32, 0x3a, 0x30, 0x30, 0x2e, 0x31, 0x32, 0x33, 0x5a}, "2023-10-17T00:12:00.123Z"},
		{[]byte{
			0x08, 0x01, 0x00, 0x00,
			0x54, 0x68, 0x65, 0x20, 0x73, 0x65, 0x6e, 0x74, 0x65, 0x6e,
			0x63, 0x65, 0x73, 0x20, 0x62, 0x65, 0x6c, 0x6f, 0x77, 0x20,
			0x61, 0x72, 0x65, 0x20, 0x70, 0x61, 0x6e, 0x67, 0x72, 0x61,
			0x6d, 0x73, 0x2e, 0x20, 0x45, 0x61, 0x63, 0x68, 0x20, 0x68,
			0x61, 0x73, 0x20, 0x61, 0x6c, 0x6c, 0x20, 0x6c, 0x65, 0x74,
			0x74, 0x65, 0x72, 0x73, 0x20, 0x6f, 0x66, 0x20, 0x74, 0x68,
			0x65, 0x20, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x62, 0x65, 0x74,
			0x2e, 0x0a, 0x41, 0x6c, 0x6c, 0x20, 0x74, 0x6f, 0x67, 0x65,
			0x74, 0x68, 0x65, 0x72, 0x2c, 0x20, 0x32, 0x36, 0x34, 0x20,
			0x62, 0x79, 0x74, 0x65, 0x73, 0x20, 0x61, 0x72, 0x65, 0x20,
			0x75, 0x73, 0x65, 0x64, 0x2e, 0x0a, 0x0a, 0x54, 0x68, 0x65,
			0x20, 0x71, 0x75, 0x69, 0x63, 0x6b, 0x20, 0x62, 0x72, 0x6f,
			0x77, 0x6e, 0x20, 0x66, 0x6f, 0x78, 0x20, 0x6a, 0x75, 0x6d,
			0x70, 0x65, 0x64, 0x20, 0x6f, 0x76, 0x65, 0x72, 0x20, 0x74,
			0x68, 0x65, 0x20, 0x6c, 0x61, 0x7a, 0x79, 0x20, 0x64, 0x6f,
			0x67, 0x2e, 0x0a, 0x50, 0x61, 0x63, 0x6b, 0x20, 0x6d, 0x79,
			0x20, 0x62, 0x6f, 0x78, 0x20, 0x77, 0x69, 0x74, 0x68, 0x20,
			0x66, 0x69, 0x76, 0x65, 0x20, 0x64, 0x6f, 0x7a, 0x65, 0x6e,
			0x20, 0x6c, 0x69, 0x71, 0x75, 0x6f, 0x72, 0x20, 0x6a, 0x75,
			0x67, 0x73, 0x2e, 0x0a, 0x41, 0x20, 0x6d, 0x61, 0x64, 0x20,
			0x62, 0x6f, 0x78, 0x65, 0x72, 0x20, 0x73, 0x68, 0x6f, 0x74,
			0x20, 0x61, 0x20, 0x71, 0x75, 0x69, 0x63, 0x6b, 0x2c, 0x20,
			0x67, 0x6c, 0x6f, 0x76, 0x65, 0x64, 0x20, 0x6a, 0x61, 0x62,
			0x20, 0x74, 0x6f, 0x20, 0x74, 0x68, 0x65, 0x20, 0x6a, 0x61,
			0x77, 0x20, 0x6f, 0x66, 0x20, 0x68, 0x69, 0x73, 0x20, 0x64,
			0x69, 0x7a, 0x7a, 0x79, 0x20, 0x6f, 0x70, 0x70, 0x6f, 0x6e,
			0x65, 0x6e, 0x74, 0x2e,
		}, `The sentences below are pangrams. Each has all letters of the alphabet.
All together, 264 bytes are used.

The quick brown fox jumped over the lazy dog.
Pack my box with five dozen liquor jugs.
A mad boxer shot a quick, gloved jab to the jaw of his dizzy opponent.`,
		},
	},
	flat.ColumnTypeBinary: {
		{[]byte{0x00, 0x00, 0x00, 0x00}, []byte{}},
		{[]byte{
			0x10, 0x00, 0x00, 0x00,
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		}, []byte{
			0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
			0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
		}},
	},
}

// testPropValueItem is an indirect reference to an item in the
// testPropValues collection.
type testPropValueItem struct {
	ct flat.ColumnType // Column type
	i  int             // Index into the column type's value array
}

func (item *testPropValueItem) value() *testPropValue {
	return &testPropValues[item.ct][item.i]
}

type testPropValueSequence struct {
	name    string
	shuffle func([]testPropValueItem)
}

const (
	minColumnType = flat.ColumnTypeByte
	maxColumnType = flat.ColumnTypeBinary
)

func (seq *testPropValueSequence) render() []testPropValueItem {
	// Collect all the values.
	var a []testPropValueItem
	for ct := minColumnType; ct <= maxColumnType; ct++ {
		for i := 0; i < len(testPropValues[ct]); i++ {
			a = append(a, testPropValueItem{ct, i})
		}
	}

	// Shuffle all the values.
	seq.shuffle(a)

	// Return the shuffled values.
	return a
}

var testPropValueSequences = []testPropValueSequence{
	{
		name:    "NoShuffle",
		shuffle: func([]testPropValueItem) { /* No op */ },
	},
	{
		name: "ShufflePredictably",
		shuffle: func(a []testPropValueItem) {
			r := rand.New(rand.NewSource(0))
			r.Shuffle(len(a), func(i, j int) {
				a[i], a[j] = a[j], a[i]
			})
		},
	},
	{
		name: "ShuffleUnpredictably",
		shuffle: func(a []testPropValueItem) {
			// This test case is technically breaking the prime
			// directive of unit testing: Thou shalt behave in a
			// strictly deterministic fashion. Given the likelihood
			// that, if this one fails, one of its deterministic
			// brothers will have failed first, it should be OK.
			r := rand.New(rand.NewSource(time.Now().Unix()))
			r.Shuffle(len(a), func(i, j int) {
				a[i], a[j] = a[j], a[i]
			})
		},
	},
}

type testPropReader func(r *PropReader) (any, error)

func bindRead[T any](f func(*PropReader) (T, error)) testPropReader {
	return func(r *PropReader) (any, error) {
		return f(r)
	}
}

var testPropReaders = map[flat.ColumnType]testPropReader{
	flat.ColumnTypeByte:   bindRead((*PropReader).ReadByte),
	flat.ColumnTypeUByte:  bindRead((*PropReader).ReadUByte),
	flat.ColumnTypeBool:   bindRead((*PropReader).ReadBool),
	flat.ColumnTypeShort:  bindRead((*PropReader).ReadShort),
	flat.ColumnTypeUShort: bindRead((*PropReader).ReadUShort),
	flat.ColumnTypeInt:    bindRead((*PropReader).ReadInt),
	flat.ColumnTypeUInt:   bindRead((*PropReader).ReadUInt),
	flat.ColumnTypeLong:   bindRead((*PropReader).ReadLong),
	flat.ColumnTypeULong:  bindRead((*PropReader).ReadULong),
	flat.ColumnTypeFloat:  bindRead((*PropReader).ReadFloat),
	flat.ColumnTypeDouble: bindRead((*PropReader).ReadDouble),
	flat.ColumnTypeString: bindRead((*PropReader).ReadString),
	flat.ColumnTypeBinary: bindRead((*PropReader).ReadBinary),
}

type errReader struct {
	error
}

func (r errReader) Read(_ []byte) (n int, err error) {
	return 0, r.error
}

func testReadColumnType(t *testing.T, c flat.ColumnType) {
	vals := testPropValues[c]
	if len(vals) < 1 {
		t.Fatalf("no test values found for column type %s", c)
	}
	f := testPropReaders[c]
	if f == nil {
		t.Fatalf("no test reader found for column type %s", c)
	}

	t.Run("Each", func(t *testing.T) {
		for i := range vals {
			t.Run(fmt.Sprintf("%d:%v", i, vals[i].de), func(t *testing.T) {
				r := NewPropReader(bytes.NewReader(vals[i].ser))

				v, err := f(r)

				assert.NoError(t, err)
				if f32, ok := vals[i].de.(float32); ok {
					require.IsType(t, reflect.TypeOf(float32(0)), reflect.TypeOf(v))
					if assertFloat(t, float64(f32), float64(v.(float32))) {
						return
					}
				} else if f64, ok := vals[i].de.(float64); ok {
					require.IsType(t, reflect.TypeOf(float64(0)), reflect.TypeOf(v))
					if assertFloat(t, f64, v.(float64)) {
						return
					}
				}
				assert.Equal(t, vals[i].de, v)
			})
		}
	})

	t.Run("All", func(t *testing.T) {
		var b bytes.Buffer
		for _, val := range vals {
			b.Write(val.ser)
		}

		r := NewPropReader(&b)
		for i := range vals {
			t.Run(fmt.Sprintf("%d:%v", i, vals[i].de), func(t *testing.T) {
				v, err := f(r)

				assert.NoError(t, err)
				if f32, ok := vals[i].de.(float32); ok {
					require.IsType(t, reflect.TypeOf(float32(0)), reflect.TypeOf(v))
					if assertFloat(t, float64(f32), float64(v.(float32))) {
						return
					}
				} else if f64, ok := vals[i].de.(float64); ok {
					require.IsType(t, reflect.TypeOf(float64(0)), reflect.TypeOf(v))
					if assertFloat(t, f64, v.(float64)) {
						return
					}
				}
				assert.Equal(t, vals[i].de, v)
			})
		}
	})

	t.Run("EOF", func(t *testing.T) {
		var b bytes.Buffer
		r := NewPropReader(&b)

		_, err := f(r)

		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("foo")
		r := NewPropReader(errReader{expectedErr})

		_, actualErr := f(r)

		assert.Same(t, expectedErr, actualErr)
	})
}

func assertFloat(t *testing.T, expected, actual float64) bool {
	if math.IsNaN(expected) {
		return assert.True(t, math.IsNaN(actual), "expected NaN but got %d", actual)
	} else {
		return false
	}
}

func TestNewPropReader(t *testing.T) {
	t.Run("Invalid Input", func(t *testing.T) {
		assert.PanicsWithValue(t, "flatgeobuf: nil reader", func() {
			NewPropReader(nil)
		})
	})
}

func TestPropReader_ReadByte(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeByte)
}

func TestPropReader_ReadUByte(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeUByte)
}

func TestPropReader_ReadBool(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeBool)
}

func TestPropReader_ReadShort(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeShort)
}

func TestPropReader_ReadUShort(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeUShort)
}

func TestPropReader_ReadInt(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeInt)
}

func TestPropReader_ReadUInt(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeUInt)
}

func TestPropReader_ReadLong(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeLong)
}

func TestPropReader_ReadULong(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeULong)
}

func TestPropReader_ReadFloat(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeFloat)
}

func TestPropReader_ReadDouble(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeDouble)
}

func TestPropReader_ReadString(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeString)
}

func TestPropReader_ReadBinary(t *testing.T) {
	testReadColumnType(t, flat.ColumnTypeBinary)

	t.Run("Error.NotEnoughData", func(t *testing.T) {
		b := []byte{0x00, 0x00, 0x00, 0x02, 0x01}
		r := NewPropReader(bytes.NewReader(b))

		val, err := r.ReadBinary()

		assert.Nil(t, val)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}

type simpleCol struct {
	name string
	ct   flat.ColumnType
}

func (col simpleCol) toColumn() flat.Column {
	bldr := flatbuffers.NewBuilder(256)
	var name flatbuffers.UOffsetT
	if col.name != "" {
		name = bldr.CreateString(col.name)
	}
	flat.ColumnStart(bldr)
	if col.name != "" {
		flat.ColumnAddName(bldr, name)
	}
	flat.ColumnAddType(bldr, col.ct)
	offset := flat.ColumnEnd(bldr)
	bldr.Finish(offset)
	return *flat.GetRootAsColumn(bldr.FinishedBytes(), 0)
}

type simpleSchema []simpleCol

func (schema simpleSchema) ColumnsLength() int {
	return len(schema)
}

func (schema simpleSchema) Columns(obj *flat.Column, i int) bool {
	if i < 0 || len(schema) <= i {
		return false
	}
	*obj = schema[i].toColumn()
	return true
}

type mockSchema struct {
	mock.Mock
}

func newMockSchema(t *testing.T) *mockSchema {
	schema := &mockSchema{}
	schema.Test(t)
	return schema
}

func (schema *mockSchema) ColumnsLength() int {
	args := schema.Called()
	return args.Int(0)
}

func (schema *mockSchema) Columns(obj *flat.Column, i int) bool {
	args := schema.Called(obj, i)
	return args.Bool(0)
}

func TestPropReader_ReadSchema(t *testing.T) {
	t.Run("Errors", func(t *testing.T) {
		t.Run("Read Schema Column Count", func(t *testing.T) {
			schema := newMockSchema(t)
			schema.
				On("ColumnsLength").
				Panic("foo")
			var b bytes.Buffer
			r := NewPropReader(&b)

			pvs, err := r.ReadSchema(schema)

			assert.Nil(t, pvs)
			assert.EqualError(t, err, "flatgeobuf: failed to read schema column count: panic: flatbuffers: foo")
			schema.AssertExpectations(t)
		})

		t.Run("Read Column Index", func(t *testing.T) {
			schema := newMockSchema(t)
			schema.
				On("ColumnsLength").
				Return(1)
			var b bytes.Buffer
			r := NewPropReader(&b)

			pvs, err := r.ReadSchema(schema)

			assert.Nil(t, pvs)
			assert.EqualError(t, err, "flatgeobuf: failed to read column index (for property 0 of 1): EOF")
			assert.ErrorIs(t, err, io.EOF)
			schema.AssertExpectations(t)
		})

		t.Run("Invalid Column Index", func(t *testing.T) {
			schema := newMockSchema(t)
			schema.
				On("ColumnsLength").
				Return(1)
			r := NewPropReader(bytes.NewReader([]byte{0x01, 0x00}))

			pvs, err := r.ReadSchema(schema)

			assert.Nil(t, pvs)
			assert.EqualError(t, err, "flatgeobuf: schema has only 1 columns, but property 0 has column index 1")
			schema.AssertExpectations(t)
		})

		t.Run("Failed to Fetch: Column Not Found", func(t *testing.T) {
			schema := newMockSchema(t)
			schema.
				On("ColumnsLength").
				Return(1)
			schema.
				On("Columns", mock.Anything, 0).
				Return(false)
			r := NewPropReader(bytes.NewReader([]byte{0x00, 0x00}))

			pvs, err := r.ReadSchema(schema)

			assert.Nil(t, pvs)
			assert.EqualError(t, err, "flatgeobuf: failed to fetch column 0 (for property 0 of 1): column not found")
			schema.AssertExpectations(t)
		})

		t.Run("Failed to Fetch: Panic", func(t *testing.T) {
			schema := newMockSchema(t)
			schema.
				On("ColumnsLength").
				Return(1)
			schema.
				On("Columns", mock.Anything, 0).
				Panic("bar")
			r := NewPropReader(bytes.NewReader([]byte{0x00, 0x00}))

			pvs, err := r.ReadSchema(schema)

			assert.Nil(t, pvs)
			assert.EqualError(t, err, "flatgeobuf: failed to fetch column 0 (for property 0 of 1): panic: flatbuffers: bar")
			schema.AssertExpectations(t)
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("Empty", func(t *testing.T) {
			schema := newMockSchema(t)
			schema.
				On("ColumnsLength").
				Return(0)
			var b bytes.Buffer
			r := NewPropReader(&b)

			pvs, err := r.ReadSchema(schema)

			assert.Empty(t, pvs)
			assert.NoError(t, err)
			schema.AssertExpectations(t)
		})

		t.Run("Each", func(t *testing.T) {
			// For each test value, generate a trivial one-element
			// schema, read the value as that schema, and validate.
			for ct := minColumnType; ct <= maxColumnType; ct++ {
				schema := simpleSchema([]simpleCol{{"foo", ct}})
				t.Run(ct.String(), func(t *testing.T) {
					vals := testPropValues[ct]
					for i, val := range vals {
						t.Run(strconv.Itoa(i), func(t *testing.T) {
							var b bytes.Buffer
							w := NewPropWriter(&b)
							_, _ = w.WriteUShort(0)
							_, _ = b.Write(val.ser)
							r := NewPropReader(&b)

							pvs, err := r.ReadSchema(schema)

							assert.NoError(t, err)
							assert.Len(t, pvs, 1)
							assert.Equal(t, pvs[0].Col.Name(), []byte("foo"))
							assert.Equal(t, pvs[0].Type, ct)
							assert.Equal(t, uint16(0), pvs[0].ColIndex)
							if f32, ok := vals[i].de.(float32); ok {
								require.IsType(t, reflect.TypeOf(float32(0)), reflect.TypeOf(pvs[0].Value))
								if assertFloat(t, float64(f32), float64(pvs[0].Value.(float32))) {
									return
								}
							} else if f64, ok := vals[i].de.(float64); ok {
								require.IsType(t, reflect.TypeOf(float64(0)), reflect.TypeOf(pvs[0].Value))
								if assertFloat(t, f64, pvs[0].Value.(float64)) {
									return
								}
							}
							assert.Equal(t, vals[i].de, pvs[0].Value)
						})
					}
				})
			}
		})

		t.Run("All", func(t *testing.T) {
			// For different combinations of all the test values,
			// generate the complete schema, write all the test values,
			// read them all as a schema, and validate.
			for _, seq := range testPropValueSequences {
				t.Run(seq.name, func(t *testing.T) {
					// Get the ordered list of values.
					a := seq.render()

					// Generate the schema.
					var schema simpleSchema
					for i, item := range a {
						name := fmt.Sprintf("Col=%d[Type=%s]", i, item.ct)
						schema = append(schema, simpleCol{name: name, ct: item.ct})
					}

					// Run print the columns both in forward and in backward
					// order.
					orientations := []struct {
						name                  string
						start, end, increment int
					}{
						{"Forward", 0, len(a), 1},
						{"Backward", len(a) - 1, -1, -1},
					}
					for _, orientation := range orientations {
						t.Run(orientation.name, func(t *testing.T) {
							// Buffer to receive writes and provide for reads
							// after.
							var b bytes.Buffer
							w := NewPropWriter(&b)

							// Write all the columns in order.
							for col := orientation.start; col != orientation.end; col += orientation.increment {
								_, err := w.WriteUShort(uint16(col)) // Column index for the property.
								require.NoError(t, err)
								f := testPropWriters[a[col].ct]
								_, err = f(w, a[col].value().de) // Property value.
								require.NoError(t, err)
							}

							// Read all the properties according to the schema.
							r := NewPropReader(&b)
							pvs, err := r.ReadSchema(schema)

							// Validate.
							assert.NoError(t, err)
							require.Len(t, pvs, len(a))
							col := orientation.start
							for i := range pvs {
								assert.Equal(t, schema[col].name, string(pvs[i].Col.Name()), "pvs[%d].Col.Name()", i)
								assert.Equal(t, a[col].ct, pvs[i].Type, "pvs[%d].Type")
								assert.Equal(t, uint16(col), pvs[i].ColIndex, "pvs[%d].ColIndex", i)

								expectedValue := a[col].value()
								actualValue := pvs[i].Value
								col += orientation.increment

								if f32, ok := expectedValue.de.(float32); ok {
									require.IsType(t, reflect.TypeOf(float32(0)), reflect.TypeOf(expectedValue))
									if assertFloat(t, float64(f32), float64(actualValue.(float32))) {
										continue
									}
								} else if f64, ok := expectedValue.de.(float64); ok {
									require.IsType(t, reflect.TypeOf(float64(0)), reflect.TypeOf(expectedValue))
									if assertFloat(t, f64, actualValue.(float64)) {
										continue
									}
								}
								assert.Equal(t, expectedValue.de, actualValue)
							}
						})
					}
				})
			}
		})
	})
}
func TestPropValue_String(t *testing.T) {
	testCases := []struct {
		name     string
		val      PropValue
		expected string
	}{
		{
			name: "Anonymous",
			val: PropValue{
				Col:      simpleCol{}.toColumn(),
				Value:    int8(5),
				ColIndex: 3,
				Type:     flat.ColumnTypeByte,
			},
			expected: "PropValue{Type:Byte,Value:5,ColIndex:3}",
		},
		{
			name: "Byte",
			val: PropValue{
				Col:      simpleCol{name: "foo"}.toColumn(),
				Value:    int8(-123),
				ColIndex: 0,
				Type:     flat.ColumnTypeByte,
			},
			expected: `PropValue{Name:"foo",Type:Byte,Value:-123,ColIndex:0}`,
		},
		{
			name: "UByte",
			val: PropValue{
				Col:      simpleCol{name: "bar"}.toColumn(),
				Value:    uint8(0xaa),
				ColIndex: 1,
				Type:     flat.ColumnTypeUByte,
			},
			expected: `PropValue{Name:"bar",Type:UByte,Value:0xaa,ColIndex:1}`,
		},
		{
			name: "Bool",
			val: PropValue{
				Col:      simpleCol{name: "baz"}.toColumn(),
				Value:    false,
				ColIndex: 2,
				Type:     flat.ColumnTypeBool,
			},
			expected: `PropValue{Name:"baz",Type:Bool,Value:false,ColIndex:2}`,
		},
		{
			name: "Short",
			val: PropValue{
				Col:      simpleCol{name: "qux"}.toColumn(),
				Value:    int16(32001),
				ColIndex: 3,
				Type:     flat.ColumnTypeShort,
			},
			expected: `PropValue{Name:"qux",Type:Short,Value:32001,ColIndex:3}`,
		},
		{
			name: "UShort",
			val: PropValue{
				Col:      simpleCol{name: "ham"}.toColumn(),
				Value:    uint16(0xfab7),
				ColIndex: 4,
				Type:     flat.ColumnTypeUShort,
			},
			expected: `PropValue{Name:"ham",Type:UShort,Value:0xfab7,ColIndex:4}`,
		},
		{
			name: "Int",
			val: PropValue{
				Col:      simpleCol{name: "eggs"}.toColumn(),
				Value:    int32(101_010_101),
				ColIndex: 5,
				Type:     flat.ColumnTypeInt,
			},
			expected: `PropValue{Name:"eggs",Type:Int,Value:101010101,ColIndex:5}`,
		},
		{
			name: "UInt",
			val: PropValue{
				Col:      simpleCol{name: "spam"}.toColumn(),
				Value:    uint32(0xbe5077ed),
				ColIndex: 6,
				Type:     flat.ColumnTypeUInt,
			},
			expected: `PropValue{Name:"spam",Type:UInt,Value:0xbe5077ed,ColIndex:6}`,
		},
		{
			name: "Long",
			val: PropValue{
				Col:      simpleCol{name: "lorem"}.toColumn(),
				Value:    int64(-88888888888888),
				ColIndex: 7,
				Type:     flat.ColumnTypeLong,
			},
			expected: `PropValue{Name:"lorem",Type:Long,Value:-88888888888888,ColIndex:7}`},
		{
			name: "ULong",
			val: PropValue{
				Col:      simpleCol{name: "ipsum"}.toColumn(),
				Value:    uint64(0x1111decafbad1111),
				ColIndex: 8,
				Type:     flat.ColumnTypeULong,
			},
			expected: `PropValue{Name:"ipsum",Type:ULong,Value:0x1111decafbad1111,ColIndex:8}`,
		},
		{
			name: "Float",
			val: PropValue{
				Col:      simpleCol{name: "dolor"}.toColumn(),
				Value:    float32(-16.5),
				ColIndex: 9,
				Type:     flat.ColumnTypeFloat,
			},
			expected: `PropValue{Name:"dolor",Type:Float,Value:-16.5,ColIndex:9}`},
		{
			name: "Double",
			val: PropValue{
				Col:      simpleCol{name: "sit"}.toColumn(),
				Value:    32.25,
				ColIndex: 10,
				Type:     flat.ColumnTypeDouble,
			},
			expected: `PropValue{Name:"sit",Type:Double,Value:32.25,ColIndex:10}`,
		},
		{
			name: "String",
			val: PropValue{
				Col:      simpleCol{name: "amet"}.toColumn(),
				Value:    "consectetur adipiscing elit, sed do eiusmod tempor",
				ColIndex: 11,
				Type:     flat.ColumnTypeString,
			},
			expected: `PropValue{Name:"amet",Type:String,Value:"consectetur adipiscing elit, sed do eiusmod tempor",ColIndex:11}`,
		},
		{
			name: "DateTime",
			val: PropValue{
				Col:      simpleCol{name: "incididunt"}.toColumn(),
				Value:    "2023-10-17T05:26:12.554Z",
				ColIndex: 12,
				Type:     flat.ColumnTypeDateTime,
			},
			expected: `PropValue{Name:"incididunt",Type:DateTime,Value:2023-10-17T05:26:12.554Z,ColIndex:12}`,
		},
		{
			name: "Binary",
			val: PropValue{
				Col:      simpleCol{name: "ut labore et dolore magna aliqua"}.toColumn(),
				Value:    []byte{0x0a, 0xb0, 0x31},
				ColIndex: 13,
				Type:     flat.ColumnTypeBinary,
			},
			expected: `PropValue{Name:"ut labore et dolore magna aliqua",Type:Binary,Value:[10 176 49],ColIndex:13}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.val.String()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}
