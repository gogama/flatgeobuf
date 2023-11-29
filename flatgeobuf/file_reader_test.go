// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/gogama/flatgeobuf/packedrtree"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileReader(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Nil Reader", func(t *testing.T) {
			assert.PanicsWithValue(t, "flatgeobuf: nil reader", func() {
				NewFileReader(nil)
			})
		})
	})
}

func TestFileReader_Header(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Header Already Called", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				_, err = r.Header()

				assert.EqualError(t, err, "flatgeobuf: Header() has already been called")
			}, false, false, "empty.fgb")
		})
	})
}

func TestFileReader_Index(t *testing.T) {
	// TODO
}

func TestFileReader_IndexSearch(t *testing.T) {
	// TODO
}

func TestFileReader_Data(t *testing.T) {
	// TODO
}

func TestFileReader_DataRem(t *testing.T) {
	// TODO
}

func TestFileReader_Rewind(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Not Seekable", func(t *testing.T) {
			r := testDataFileReader(t, false, "empty.fgb")
			_, err := r.Header()
			require.NoError(t, err)

			err = r.Rewind()

			assert.EqualError(t, err, "flatgeobuf: can't rewind: reader is not an io.Seeker")
		})

		t.Run("Header Not Called", func(t *testing.T) {
			r := NewFileReader(&bytes.Buffer{})

			err := r.Rewind()

			assert.EqualError(t, err, "flatgeobuf: must call Header()")
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("State: After Header", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				err = r.Rewind()

				assert.NoError(t, err)
			}, true, true)
		})

		t.Run("State: After Index", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				index1, err1 := r.Index()
				if index1 != nil {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err1, ErrNoIndex)
				}

				err = r.Rewind()
				assert.NoError(t, err)

				index2, err2 := r.Index()
				assert.Same(t, index1, index2) // Cached index or nil
				assert.Equal(t, err1, err2)
			}, true, true)
		})

		t.Run("State: In Data", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader) {
				hdr, err := r.Header()
				require.NoError(t, err)
				p := make([]flat.Feature, 1)
				m, err1 := r.Data(p)
				if err1 != nil {
					require.ErrorIs(t, err1, io.EOF)
					require.Equal(t, 0, m)
					require.Equal(t, hdr.FeaturesCount(), uint64(0))
				} else {
					require.Equal(t, 1, m)
				}

				err = r.Rewind()
				assert.NoError(t, err)

				q := make([]flat.Feature, 1)
				n, err2 := r.Data(q)
				assert.Equal(t, m, n)
				assert.Equal(t, err1, err2)
			}, true, true)
		})

		t.Run("State: EOF", func(t *testing.T) {
			t.Run("After DataRem", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p, err := r.DataRem()
					require.NoError(t, err)

					err = r.Rewind()
					assert.NoError(t, err)

					q, err := r.DataRem()
					assert.NoError(t, err)
					assert.Equal(t, p, q)
				}, true, true)
			})

			t.Run("After IndexSearch", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					if hdr.IndexNodeSize() == 0 {
						t.Skip("No index")
					}
					_, err = r.IndexSearch(packedrtree.Box{XMin: -180, YMin: -90, XMax: 180, YMax: 90})
					require.NoError(t, err)

					err = r.Rewind()
					assert.NoError(t, err)
				}, true, true)
			})
		})

		t.Run("Repeated", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				n := 10
				for i := 0; i < n; i++ {
					t.Run(strconv.Itoa(i), func(t *testing.T) {
						if i%2 == 0 {
							_, _ = r.Index()
						}
						if i%3 == 0 {
							_, err := r.DataRem()
							require.NoError(t, err)
						}

						err := r.Rewind()
						assert.NoError(t, err)
					})
				}
			}, true, true)
		})
	})
}

type mockReadCloser struct {
	mock.Mock
}

func newMockReadCloser(t *testing.T) *mockReadCloser {
	m := &mockReadCloser{}
	m.Test(t)
	return m
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *mockReadCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestFileReader_Close(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Already Closed", func(t *testing.T) {
			var b bytes.Buffer
			r := NewFileReader(&b)
			err := r.Close()
			require.NoError(t, err)

			err = r.Close()

			assert.Same(t, err, ErrClosed)
		})

		t.Run("Has io.Closer Error", func(t *testing.T) {
			m := newMockReadCloser(t)
			r := NewFileReader(m)
			expectedErr := errors.New("foo")
			m.On("Close").Return(expectedErr)

			err := r.Close()

			assert.Same(t, err, expectedErr)
			m.AssertExpectations(t)
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("Has io.Closer", func(t *testing.T) {
			m := newMockReadCloser(t)
			r := NewFileReader(m)
			m.On("Close").Return(nil)

			err := r.Close()

			assert.NoError(t, err)
			m.AssertExpectations(t)
		})

		t.Run("No io.Closer", func(t *testing.T) {
			var b bytes.Buffer
			r := NewFileReader(&b)

			err := r.Close()

			assert.NoError(t, err)
		})
	})
}
