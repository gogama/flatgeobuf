// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewFileWriter(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Nil Writer", func(t *testing.T) {
			assert.PanicsWithValue(t, "flatgeobuf: nil writer", func() {
				NewFileWriter(nil)
			})
		})
	})
}

func TestFileWriter_Header(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Header Already Called", func(t *testing.T) {
			// TODO
		})

		t.Run("Nil Header", func(t *testing.T) {
			// TODO
		})

		t.Run("Corrupt Header: Failed to Get Feature Count", func(t *testing.T) {
			// TODO
		})
	})
}

func TestFileWriter_Index(t *testing.T) {

}

func TestFileWriter_IndexData(t *testing.T) {

}

func TestFileWriter_Data(t *testing.T) {

}

type mockWriteCloser struct {
	mock.Mock
}

func newMockWriteCloser(t *testing.T) *mockWriteCloser {
	m := &mockWriteCloser{}
	m.Test(t)
	return m
}

func (m *mockWriteCloser) Write(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *mockWriteCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestFileWriter_Close(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Already Closed", func(t *testing.T) {
			var b bytes.Buffer
			w := NewFileWriter(&b)
			err := w.Close()
			require.NoError(t, err)

			err = w.Close()

			assert.Same(t, err, ErrClosed)
		})

		t.Run("Has io.Closer Error", func(t *testing.T) {
			m := newMockWriteCloser(t)
			w := NewFileWriter(m)
			expectedErr := errors.New("foo")
			m.On("Close").Return(expectedErr)

			err := w.Close()

			assert.Same(t, err, expectedErr)
			m.AssertExpectations(t)
		})

		t.Run("Truncated File", func(t *testing.T) {
			hdr := (&mockHeader{
				name:          stringPtr(t.Name()),
				featuresCount: uint64(1),
			}).buildAsTable()
			var b bytes.Buffer
			w := NewFileWriter(&b)
			n, err := w.Header(hdr)
			require.NoError(t, err)
			require.Greater(t, n, len(hdr.Table().Bytes))

			err = w.Close()

			assert.EqualError(t, err, "flatgeobuf: truncated file: only wrote 0 of 1 header-indicated features")

			err = w.Close()

			assert.Same(t, err, ErrClosed)
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("Has io.Closer", func(t *testing.T) {
			m := newMockWriteCloser(t)
			w := NewFileWriter(m)
			m.On("Close").Return(nil)

			err := w.Close()

			assert.NoError(t, err)
			m.AssertExpectations(t)
		})

		t.Run("No io.Closer", func(t *testing.T) {
			var b bytes.Buffer
			w := NewFileWriter(&b)

			err := w.Close()

			assert.NoError(t, err)
		})
	})
}
