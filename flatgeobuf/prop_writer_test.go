// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPropWriter func(*PropWriter, any) (int, error)

func bindWrite[T any](f func(*PropWriter, T) (int, error)) testPropWriter {
	return func(w *PropWriter, v any) (int, error) {
		return f(w, v.(T))
	}
}

var testPropWriters = map[flat.ColumnType]testPropWriter{
	flat.ColumnTypeByte:   bindWrite((*PropWriter).WriteByte),
	flat.ColumnTypeUByte:  bindWrite((*PropWriter).WriteUByte),
	flat.ColumnTypeBool:   bindWrite((*PropWriter).WriteBool),
	flat.ColumnTypeShort:  bindWrite((*PropWriter).WriteShort),
	flat.ColumnTypeUShort: bindWrite((*PropWriter).WriteUShort),
	flat.ColumnTypeInt:    bindWrite((*PropWriter).WriteInt),
	flat.ColumnTypeUInt:   bindWrite((*PropWriter).WriteUInt),
	flat.ColumnTypeLong:   bindWrite((*PropWriter).WriteLong),
	flat.ColumnTypeULong:  bindWrite((*PropWriter).WriteULong),
	flat.ColumnTypeFloat:  bindWrite((*PropWriter).WriteFloat),
	flat.ColumnTypeDouble: bindWrite((*PropWriter).WriteDouble),
	flat.ColumnTypeString: bindWrite((*PropWriter).WriteString),
	flat.ColumnTypeBinary: bindWrite((*PropWriter).WriteBinary),
}

type errWriter struct {
	error
}

func (w errWriter) Write(_ []byte) (n int, err error) {
	return 0, w.error
}

func testWriteColumnType(t *testing.T, c flat.ColumnType) {
	vals := testPropValues[c]
	if len(vals) < 1 {
		t.Fatalf("no test values found for column type %s", c)
	}
	f := testPropWriters[c]
	if f == nil {
		t.Fatalf("no test reader found for column type %s", c)
	}

	t.Run("Each", func(t *testing.T) {
		for i := range vals {
			t.Run(fmt.Sprintf("%d:%v", i, vals[i].de), func(t *testing.T) {
				var b bytes.Buffer
				w := NewPropWriter(&b)

				n, err := f(w, vals[i].de)

				assert.NoError(t, err)
				assert.Equal(t, len(vals[i].ser), n)
				assert.Equal(t, vals[i].ser, b.Bytes())
			})
		}
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("foo")

		for i := range vals {
			t.Run(fmt.Sprintf("%d:%v", i, vals[i].de), func(t *testing.T) {
				w := NewPropWriter(errWriter{expectedErr})

				n, actualErr := f(w, vals[i].de)

				assert.Equal(t, 0, n)
				assert.Same(t, expectedErr, actualErr)
			})
		}
	})
}

func TestNewPropWriter(t *testing.T) {
	t.Run("Invalid Input", func(t *testing.T) {
		assert.PanicsWithValue(t, "flatgeobuf: nil writer", func() {
			NewPropWriter(nil)
		})
	})
}

func TestPropWriter_WriteByte(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeByte)
}

func TestPropWriter_WriteUByte(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeUByte)
}

func TestPropWriter_WriteBool(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeBool)
}

func TestPropWriter_WriteShort(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeShort)
}

func TestPropWriter_WriteUShort(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeUShort)
}

func TestPropWriter_WriteInt(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeInt)
}

func TestPropWriter_WriteUInt(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeUInt)
}

func TestPropWriter_WriteLong(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeLong)
}

func TestPropWriter_WriteULong(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeULong)
}

func TestPropWriter_WriteFloat(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeFloat)
}

func TestPropWriter_WriteDouble(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeDouble)
}

func TestPropWriter_WriteString(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeString)
}

func TestPropWriter_WriteBinary(t *testing.T) {
	testWriteColumnType(t, flat.ColumnTypeBinary)
}

func TestPropRoundtrip(t *testing.T) {
	// The purpose of this function is to verify that if you write all the test
	// cases, you can read them back.
	for _, seq := range testPropValueSequences {
		t.Run(seq.name, func(t *testing.T) {
			// Get the ordered list of values.
			a := seq.render()

			// Buffer to receive writes and provide for reads after.
			var b bytes.Buffer

			// Write all the values.
			t.Run("Write", func(t *testing.T) {
				w := NewPropWriter(&b)
				for i, item := range a {
					t.Run(fmt.Sprintf("%d:%s[%d]", i, item.ct, item.i), func(t *testing.T) {
						f := testPropWriters[item.ct]
						val := testPropValues[item.ct][item.i]
						n, err := f(w, val.de)

						assert.NoError(t, err)
						assert.Equal(t, len(val.ser), n)
					})
				}
			})

			// Read all the values.
			t.Run("Read", func(t *testing.T) {
				r := NewPropReader(&b)
				for i, x := range a {
					t.Run(fmt.Sprintf("%d:%d[%d]", i, x.ct, x.i), func(t *testing.T) {
						f := testPropReaders[x.ct]
						val := testPropValues[x.ct][x.i]
						v, err := f(r)

						assert.NoError(t, err)
						if f32, ok := val.de.(float32); ok {
							require.IsType(t, reflect.TypeOf(float32(0)), reflect.TypeOf(v))
							if assertFloat(t, float64(f32), float64(v.(float32))) {
								return
							}
						} else if f64, ok := val.de.(float64); ok {
							require.IsType(t, reflect.TypeOf(float64(0)), reflect.TypeOf(v))
							if assertFloat(t, f64, v.(float64)) {
								return
							}
						}
						assert.Equal(t, val.de, v)
					})
				}
			})
		})
	}
}
