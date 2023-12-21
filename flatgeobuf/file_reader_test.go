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
	"strconv"
	"testing"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	"github.com/gogama/flatgeobuf/packedrtree"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				_, err = r.Header()

				assert.EqualError(t, err, "flatgeobuf: Header() has already been called")
			}, notSeekable, "empty.fgb")
		})

		t.Run("Failed to Read Magic Number", func(t *testing.T) {
			r := NewFileReader(&bytes.Buffer{})

			hdr, err := r.Header()

			assert.Nil(t, hdr)
			assert.EqualError(t, err, "flatgeobuf: failed to read magic number: EOF")
			assert.ErrorIs(t, err, io.EOF)
		})

		t.Run("Unsupported Major Version", func(t *testing.T) {
			r := NewFileReader(bytes.NewReader(make([]byte, magicLen)))

			hdr, err := r.Header()

			assert.Nil(t, hdr)
			assert.EqualError(t, err, "flatgeobuf: failed to read magic number: flatgeobuf: invalid magic number")
		})

		t.Run("Header Length Read Error", func(t *testing.T) {
			b := make([]byte, magicLen+1)
			copy(b, magic[:])
			r := NewFileReader(bytes.NewReader(b))

			hdr, err := r.Header()

			assert.Nil(t, hdr)
			assert.EqualError(t, err, "flatgeobuf: header length read error: unexpected EOF")
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		})

		t.Run("Header Length too Small", func(t *testing.T) {
			b := make([]byte, magicLen+flatbuffers.SizeUint32)
			copy(b, magic[:])
			r := NewFileReader(bytes.NewReader(b))

			hdr, err := r.Header()

			assert.Nil(t, hdr)
			assert.EqualError(t, err, "flatgeobuf: header length 0 not big enough for FlatBuffer uoffset_t")
		})

		t.Run("Failed to Read Header", func(t *testing.T) {
			b := make([]byte, magicLen+flatbuffers.SizeUint32+1)
			copy(b, magic[:])
			flatbuffers.WriteUint32(b[magicLen:], 100)

			r := NewFileReader(bytes.NewReader(b))

			hdr, err := r.Header()

			assert.Nil(t, hdr)
			assert.EqualError(t, err, "flatgeobuf: failed to read header table (len=100): unexpected EOF")
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		})

		t.Run("Feature Count Overflows Max Int", func(t *testing.T) {
			featuresCount := uint64(math.MaxInt) + 1
			mh := mockHeader{
				featuresCount: featuresCount,
			}
			var b bytes.Buffer
			_, _ = b.Write(magic[:])
			_, _ = b.Write(mh.buildAsBytes())
			r := NewFileReader(&b)

			hdr, err1 := r.Header()

			assert.ErrorContains(t, err1, "flatgeobuf: header feature count")
			assert.ErrorContains(t, err1, " overflows limit of ")
			assert.ErrorContains(t, err1, " features")
			require.NotNil(t, hdr)
			assert.Equal(t, featuresCount, hdr.FeaturesCount())

			index, err2 := r.Index()

			assert.Nil(t, index)
			assert.Same(t, err1, err2)

			p, err3 := r.IndexSearch(packedrtree.Box{XMax: 100, YMax: 100})

			assert.Nil(t, p)
			assert.Same(t, err1, err3)

			q := make([]flat.Feature, 1)
			n, err4 := r.Data(q)

			assert.Equal(t, 0, n)
			assert.Same(t, err1, err4)

			s, err5 := r.DataRem()

			assert.Nil(t, s)
			assert.Same(t, err1, err5)
		})

		t.Run("Invalid Index Node Size", func(t *testing.T) {
			indexNodeSize := uint16(1)
			mh := mockHeader{
				indexNodeSize: uint16Ptr(indexNodeSize),
			}
			var b bytes.Buffer
			_, _ = b.Write(magic[:])
			_, _ = b.Write(mh.buildAsBytes())
			r := NewFileReader(&b)

			hdr, err1 := r.Header()

			assert.EqualError(t, err1, "flatgeobuf: header index node size 1 not allowed")
			require.NotNil(t, hdr)
			assert.Equal(t, indexNodeSize, hdr.IndexNodeSize())

			index, err2 := r.Index()

			assert.Nil(t, index)
			assert.Same(t, err1, err2)

			p, err3 := r.IndexSearch(packedrtree.Box{XMax: 100, YMax: 100})

			assert.Nil(t, p)
			assert.Same(t, err1, err3)

			q := make([]flat.Feature, 1)
			n, err4 := r.Data(q)

			assert.Equal(t, 0, n)
			assert.Same(t, err1, err4)

			s, err5 := r.DataRem()

			assert.Nil(t, s)
			assert.Same(t, err1, err5)
		})

		t.Run("Failed to Save Index Offset", func(t *testing.T) {
			mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
				expectedErr := errors.New("foo")
				mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{0, io.SeekCurrent, 0, expectedErr})
				r := NewFileReader(mr)

				hdr, err := r.Header()

				assert.Nil(t, hdr)
				assert.EqualError(t, err, "flatgeobuf: failed to query index offset: foo")
				assert.ErrorIs(t, err, expectedErr)
				mr.verify()
			}, "one_feature_with_index")
		})

		t.Run("Failed to Save Data Offset", func(t *testing.T) {
			mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
				expectedErr := errors.New("bar")
				mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{0, io.SeekCurrent, 1, expectedErr})
				r := NewFileReader(mr)

				hdr, err := r.Header()

				assert.Nil(t, hdr)
				assert.EqualError(t, err, "flatgeobuf: failed to query data offset: bar")
				assert.ErrorIs(t, err, expectedErr)
				mr.verify()
			}, "empty")
		})
	})

	t.Run("Success", func(t *testing.T) {
		testCases := []struct {
			name string
			mh   mockHeader
		}{
			{
				name: "Zero",
				mh: mockHeader{
					indexNodeSize: uint16Ptr(0),
				},
			},
			{
				name: "All Set",
				mh: mockHeader{
					name:         stringPtr("foo"),
					envelope:     []float64{0, 1, -1, 100},
					geometryType: flat.GeometryTypeCurve,
					hasZ:         true,
					hasM:         true,
					hasT:         true,
					hasTM:        true,
					columns: []mockColumn{
						{
							name:        "bar",
							columnType:  flat.ColumnTypeUByte,
							title:       stringPtr("baz"),
							description: stringPtr("qux"),
							width:       100,
							precision:   -50,
							scale:       -1,
							nullable:    true,
							unique:      true,
							primaryKey:  true,
							metadata:    stringPtr(""),
						},
					},
					featuresCount: 1001,
					indexNodeSize: uint16Ptr(27),
					crs: &mockCRS{
						org:         stringPtr("ham"),
						code:        13,
						name:        stringPtr("eggs"),
						description: stringPtr("spam"),
						wkt:         stringPtr("this ain't geometry!"),
						codeString:  stringPtr("code it up"),
					},
					title:       stringPtr("the quick brown fox"),
					description: stringPtr("jumped over the lazy dog"),
					metadata:    stringPtr("hello, world!"),
				},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				var b bytes.Buffer
				_, _ = b.Write(magic[:])
				_, _ = b.Write(testCase.mh.buildAsBytes())
				r := NewFileReader(&b)

				hdr, err := r.Header()
				assert.NoError(t, err)
				require.NotNil(t, hdr)

				mh := mockHeaderFromFlatBufferTable(hdr)
				assert.Equal(t, testCase.mh, mh)
			})
		}
	})
}

func TestFileReader_Index(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Unexpected State", func(t *testing.T) {
			t.Run("Before: Header", func(t *testing.T) {
				r := NewFileReader(&bytes.Buffer{})

				index, err := r.Index()

				assert.EqualError(t, err, "flatgeobuf: must call Header()")
				assert.Nil(t, index)
			})

			t.Run("After: Index", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					index, err := r.Index()
					require.NoError(t, err)
					require.NotNil(t, index)

					index, err = r.Index()

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, index)
				}, skipNoIndex|seekable|notSeekable)
			})

			t.Run("After: IndexSearch", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p, err := r.IndexSearch(packedrtree.Box{XMin: -50, YMin: -50, XMax: 50, YMax: 50})
					require.NoError(t, err)
					require.NotNil(t, p)

					index, err := r.Index()

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, index)
				}, skipNoIndex|seekable|notSeekable)
			})

			t.Run("After: Data", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p := make([]flat.Feature, 100)
					n, err := r.Data(p)
					if !(n == 0 && errors.Is(err, io.EOF)) {
						require.NoError(t, err)
					}

					index, err := r.Index()

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, index)
				}, seekable|notSeekable)
			})

			t.Run("After: DataRem", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p, err := r.DataRem()
					if p != nil || !errors.Is(err, io.EOF) {
						require.NoError(t, err)
					}

					index, err := r.Index()

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, index)
				}, seekable|notSeekable)
			})
		})

		t.Run("Usage Error", func(t *testing.T) {
			t.Run("No Index", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)

					index, err := r.Index()

					assert.Nil(t, index)
					assert.Error(t, err, "flatgeobuf: no index")
				}, notSeekable, "empty.fgb", "unknown_feature_count.fgb")
			})
		})

		t.Run("Underlying Reader Error", func(t *testing.T) {
			t.Run("Failed to Read Index", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					mr := newMockDataBytesReader(t, mf, nil)
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					index, err := r.Index()

					assert.EqualError(t, err, "flatgeobuf: failed to read index: packedrtree: failed to read index bytes: EOF")
					assert.ErrorIs(t, err, io.EOF)
					assert.Nil(t, index)
					mr.verify()
				}, "truncated_index")
			})

			t.Run("Failed to Save Data Offset", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("baz")
					mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{0, io.SeekCurrent, 1, expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					index, err := r.Index()

					assert.EqualError(t, err, "flatgeobuf: failed to query data offset: baz")
					assert.ErrorIs(t, err, expectedErr)
					assert.Nil(t, index)
					mr.verify()
				}, "one_feature_with_index")
			})

			t.Run("Failed to Seek Past Cached Index", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("you shall not pass")
					mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{mf.dataOffsets[0], io.SeekStart, 0, expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)
					index, err := r.Index()
					require.NotNil(t, index)
					require.NoError(t, err)
					err = r.Rewind()
					require.NoError(t, err)

					index, err = r.Index()

					assert.EqualError(t, err, "flatgeobuf: failed to seek past cached index: you shall not pass")
					assert.ErrorIs(t, err, expectedErr)
					assert.Nil(t, index)
					mr.verify()
				}, "one_feature_with_index")
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
			hdr, err := r.Header()
			require.NoError(t, err)
			require.NotNil(t, hdr)

			index, err := r.Index()

			assert.NotNil(t, index)
			assert.NoError(t, err)
			assert.Equal(t, hdr.FeaturesCount(), uint64(index.NumRefs()))
			assert.Equal(t, hdr.IndexNodeSize(), index.NodeSize())
		}, notSeekable|skipNoIndex)
	})
}

func TestFileReader_IndexSearch(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Unexpected State", func(t *testing.T) {
			t.Run("Before: Header", func(t *testing.T) {
				r := NewFileReader(&bytes.Buffer{})

				p, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})

				assert.EqualError(t, err, "flatgeobuf: must call Header()")
				assert.Nil(t, p)
			})

			t.Run("After: Index", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					index, err := r.Index()
					require.NoError(t, err)
					require.NotNil(t, index)

					p, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, p)
				}, skipNoIndex|seekable|notSeekable)
			})

			t.Run("After: IndexSearch", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})
					require.NoError(t, err)
					require.NotNil(t, p)

					q, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, q)
				}, skipNoIndex|seekable|notSeekable)
			})

			t.Run("After: Data", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p := make([]flat.Feature, 100)
					n, err := r.Data(p)
					if !(n == 0 && errors.Is(err, io.EOF)) {
						require.NoError(t, err)
					}

					q, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, q)
				}, seekable|notSeekable)
			})

			t.Run("After: DataRem", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p, err := r.DataRem()
					if p != nil || !errors.Is(err, io.EOF) {
						require.NoError(t, err)
					}

					q, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})

					assert.ErrorContains(t, err, "flatgeobuf: read position is past index")
					assert.Nil(t, q)
				}, seekable|notSeekable)
			})
		})

		t.Run("Usage Error", func(t *testing.T) {
			t.Run("No Index", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)

					p, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})

					assert.Nil(t, p)
					assert.Error(t, err, "flatgeobuf: no index")
				}, notSeekable, "empty.fgb", "unknown_feature_count.fgb")
			})
		})

		t.Run("Underlying Reader Error", func(t *testing.T) {
			t.Run("Failed to Seek Past Index", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("no seeking allowed")
					mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{mf.dataOffsets[0], io.SeekStart, 0, expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)
					index, err := r.Index()
					require.NotNil(t, index)
					require.NoError(t, err)
					err = r.Rewind()
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.EmptyBox)

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: failed to seek past index: no seeking allowed")
					assert.ErrorIs(t, err, expectedErr)
					mr.verify()
				}, "one_feature_with_index")
			})

			t.Run("Failed to Seek to Index Start", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("seek and ye shall not find")
					mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{mf.indexOffset, io.SeekStart, 0, expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.EmptyBox)

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: failed to seek to index start: seek and ye shall not find")
					assert.ErrorIs(t, err, expectedErr)
					mr.verify()
				}, "one_feature_with_index")
			})

			t.Run("Failed to Execute the Seek Search", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					mr := newMockDataBytesReadSeeker(t, mf, nil, nil)
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.EmptyBox)

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: failed to seek-search index: packedrtree: failed to read nodes [0..1), rel. offset 0: EOF")
					assert.ErrorIs(t, err, io.EOF)
					mr.verify()
				}, "truncated_index")
			})

			t.Run("Failed to Cache the Index", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					mr := newMockDataBytesReader(t, mf, nil)
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.EmptyBox)

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: failed to read index: packedrtree: failed to read index bytes: EOF")
					assert.ErrorIs(t, err, io.EOF)
					mr.verify()
				}, "truncated_index")
			})

			t.Run("Failed to Save Data Offset", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("un-seekable data ahead")
					mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{0, io.SeekCurrent, 2, expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.EmptyBox)

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: failed to query data offset: un-seekable data ahead")
					assert.ErrorIs(t, err, expectedErr)
					mr.verify()
				}, "one_feature_with_index")
			})

			t.Run("Failed to Skip to Feature", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("it's a bug, not a feature")
					mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{mf.dataOffsets[2] - mf.dataOffsets[0], io.SeekCurrent, 0, expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.Box{XMin: 1, YMin: 1, XMax: 1, YMax: 1})

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: failed to skip to feature 2 (data offset 160) for search result 0: it's a bug, not a feature")
					assert.ErrorIs(t, err, expectedErr)
					mr.verify()
				}, "four_points_in_quadrants")
			})

			t.Run("Data Section Ends before Expected Feature", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					mr := newMockDataBytesReadSeeker(t, mf, nil, nil)
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.Box{})

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: data section ends before feature[0]: unexpected EOF")
					assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
					mr.verify()
				}, "truncated_data")
			})

			t.Run("Feature Read Error", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("some kind of reading problem")
					mr := newMockDataBytesReader(t, mf, &mockDataReadError{pos: mf.dataOffsets[0] + 1, err: expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)

					p, err := r.IndexSearch(packedrtree.Box{XMin: -1, YMin: -1})

					assert.Nil(t, p)
					assert.EqualError(t, err, "flatgeobuf: feature[0] length read error (offset 0): some kind of reading problem")
					assert.ErrorIs(t, err, expectedErr)
					mr.verify()
				}, "one_feature_with_index")
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("Reuse Cached Index", func(t *testing.T) {
			mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
				mr := newMockDataBytesReadSeeker(t, mf, nil, nil)
				r := NewFileReader(mr)
				hdr, err := r.Header()
				require.NotNil(t, hdr)
				require.NoError(t, err)
				index, err := r.Index()
				require.NotNil(t, index)
				require.NoError(t, err)
				err = r.Rewind()
				require.NoError(t, err)

				p, err := r.IndexSearch(packedrtree.Box{XMin: -1, YMin: -1, XMax: 0, YMax: 0})

				assert.Len(t, p, 1)
				mr.verify()
			}, "one_feature_with_index")
		})

		t.Run("Expected Feature Count", func(t *testing.T) {
			actualResults := make(map[string][]flat.Feature)
			expectedResults := []struct {
				filename string
				n        int
			}{
				{
					filename: "alldatatypes.fgb",
					n:        1,
				},
				{
					filename: "countries.fgb",
					n:        12,
				},
				{
					filename: "UScounties.fgb",
					n:        0,
				},
			}

			t.Run("Search", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, filename string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)

					p, err := r.IndexSearch(packedrtree.Box{XMin: -100, YMin: -100})

					assert.NoError(t, err)
					assert.NotNil(t, p)
					actualResults[filename] = p
				}, notSeekable|skipNoIndex)
			})

			t.Run("Verify", func(t *testing.T) {
				for _, expectedResult := range expectedResults {
					t.Run(expectedResult.filename, func(t *testing.T) {
						assert.Equal(t, len(actualResults[expectedResult.filename]), expectedResult.n)
					})
				}
			})
		})
	})
}

func TestFileReader_Data(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Unexpected State", func(t *testing.T) {
			t.Run("Before: Header", func(t *testing.T) {
				r := NewFileReader(&bytes.Buffer{})
				p := make([]flat.Feature, 1)

				n, err := r.Data(p)

				assert.Equal(t, 0, n)
				assert.EqualError(t, err, "flatgeobuf: must call Header()")
			})

			t.Run("Already in Error State", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					r := NewFileReader(newMockDataBytesReader(t, mf, nil))
					hdr, err1 := r.Header()
					require.EqualError(t, err1, "flatgeobuf: header length read error: EOF")
					require.Nil(t, hdr)
					p := make([]flat.Feature, 1)

					n, err2 := r.Data(p)

					assert.Equal(t, 0, n)
					assert.Same(t, err1, err2)
				}, "truncated_header")
			})
		})

		t.Run("Underlying Reader Error", func(t *testing.T) {
			t.Run("Skipping Index", func(t *testing.T) {
				t.Run("Seek to Data", func(t *testing.T) {
					for _, name := range []string{"empty", "one_feature_with_index"} {
						mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
							expectedErr := errors.New("pow")
							mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{mf.dataOffsets[0], io.SeekStart, 0, expectedErr})
							r := NewFileReader(mr)
							hdr, err := r.Header()
							require.NotNil(t, hdr)
							require.NoError(t, err)
							p := make([]flat.Feature, 1)

							n, err := r.Data(p)

							assert.Equal(t, 0, n)
							assert.EqualError(t, err, "flatgeobuf: failed to seek to data section: pow")
							assert.ErrorIs(t, err, expectedErr)
						}, name)
					}
				})

				t.Run("Index Size Overflow", func(t *testing.T) {
					for _, seeker := range []bool{false, true} {
						t.Run(fmt.Sprintf("Seekable: %t", seeker), func(t *testing.T) {
							mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
								var mr io.Reader
								if seeker {
									mr = newMockDataBytesReadSeeker(t, mf, nil, nil)
								} else {
									mr = newMockDataBytesReader(t, mf, nil)
								}
								r := NewFileReader(mr)
								hdr, err := r.Header()
								require.NotNil(t, hdr)
								if hdr.FeaturesCount() > uint64(math.MaxInt) {
									require.ErrorContains(t, err, "header feature count ")
									require.ErrorContains(t, err, " limit of ")
									require.ErrorContains(t, err, " features")
								} else {
									p := make([]flat.Feature, 1)
									n, err := r.Data(p)
									assert.Equal(t, 0, n)
									assert.EqualError(t, err, "flatgeobuf: failed to calculate index size: packedrtree: total node count overflows int")
								}
							}, "index_size_overflow")
						})
					}
				})

				t.Run("Read Past Index", func(t *testing.T) {
					mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
						mr := newMockDataBytesReader(t, mf, nil)
						r := NewFileReader(mr)
						hdr, err := r.Header()
						require.NoError(t, err)
						require.NotNil(t, hdr)
						p := make([]flat.Feature, 1)

						n, err := r.Data(p)

						assert.Equal(t, 0, n)
						assert.EqualError(t, err, "flatgeobuf: failed to read past index: EOF")
						assert.ErrorIs(t, err, io.EOF)
					}, "truncated_index")
				})
			})

			t.Run("Feature Read Error", func(t *testing.T) {
				t.Run("Feature Length Read Error", func(t *testing.T) {
					files := []string{"one_feature_with_index", "one_feature_no_index", "one_feature_no_index_unknown_count"}

					for _, seeker := range []bool{false, true} {
						t.Run(fmt.Sprintf("Seekable: %t", seeker), func(t *testing.T) {
							for _, file := range files {
								mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
									expectedErr := errors.New("problem reading the feature length")
									readErr := &mockDataReadError{mf.dataOffsets[0] + 1, expectedErr}
									var r *FileReader
									if seeker {
										mr := newMockDataBytesReadSeeker(t, mf, readErr, nil)
										defer mr.verify()
										r = NewFileReader(mr)
									} else {
										mr := newMockDataBytesReader(t, mf, readErr)
										defer mr.verify()
										r = NewFileReader(mr)
									}
									hdr, err := r.Header()
									require.NoError(t, err)
									require.NotNil(t, hdr)
									p := make([]flat.Feature, 1)

									n, err := r.Data(p)

									assert.Equal(t, 0, n)
									assert.EqualError(t, err, "flatgeobuf: feature[0] length read error (offset 0): problem reading the feature length")
									assert.ErrorIs(t, err, expectedErr)
								}, file)
							}
						})
					}
				})

				t.Run("Feature Length Not Big Enough for FlatBuffer uoffset_t", func(t *testing.T) {
					for _, seeker := range []bool{false, true} {
						t.Run(fmt.Sprintf("Seekable: %t", seeker), func(t *testing.T) {
							mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
								var mr io.Reader
								if seeker {
									mr = newMockDataBytesReadSeeker(t, mf, nil, nil)
								} else {
									mr = newMockDataBytesReader(t, mf, nil)
								}
								r := NewFileReader(mr)
								hdr, err := r.Header()
								require.NotNil(t, hdr)
								p := make([]flat.Feature, 1)

								n, err := r.Data(p)

								assert.Equal(t, 0, n)
								assert.EqualError(t, err, "flatgeobuf: feature[0] length 0 not big enough for FlatBuffer uoffset_t (offset 0)")
							}, "feature_length_too_small")
						})
					}
				})

				t.Run("Failed to Read Feature", func(t *testing.T) {
					files := []string{"one_feature_with_index", "one_feature_no_index", "one_feature_no_index_unknown_count"}

					for _, seeker := range []bool{false, true} {
						t.Run(fmt.Sprintf("Seekable: %t", seeker), func(t *testing.T) {
							for _, file := range files {
								mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
									expectedErr := errors.New("problem reading the feature data")
									readErr := &mockDataReadError{mf.dataOffsets[0] + flatbuffers.SizeUint32 + 1, expectedErr}
									var r *FileReader
									if seeker {
										mr := newMockDataBytesReadSeeker(t, mf, readErr, nil)
										defer mr.verify()
										r = NewFileReader(mr)
									} else {
										mr := newMockDataBytesReader(t, mf, readErr)
										defer mr.verify()
										r = NewFileReader(mr)
									}
									hdr, err := r.Header()
									require.NoError(t, err)
									require.NotNil(t, hdr)
									p := make([]flat.Feature, 1)

									n, err := r.Data(p)

									assert.Equal(t, 0, n)
									assert.EqualError(t, err, "flatgeobuf: failed to read feature[0] (offset=0, len=92): problem reading the feature data")
									assert.ErrorIs(t, err, expectedErr)
								}, file)
							}
						})
					}
				})
			})
		})

		t.Run("EOF", func(t *testing.T) {
			t.Run("Empty File", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p := make([]flat.Feature, 1)

					n, err := r.Data(p)

					assert.Equal(t, 0, n)
					assert.ErrorIs(t, err, io.EOF)
				}, notSeekable, "empty.fgb")
			})

			t.Run("Read to End", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					_, _ = r.DataRem()
					q := make([]flat.Feature, 1)

					n, err := r.Data(q)

					assert.Equal(t, 0, n)
					assert.ErrorIs(t, err, io.EOF)
				}, notSeekable)
			})
		})

		t.Run("Corrupt File", func(t *testing.T) {
			t.Run("Corrupted Feature", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					mr := newMockDataBytesReadSeeker(t, mf, nil, nil)
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)
					p := make([]flat.Feature, 1)
					n, err := r.Data(p)
					require.NoError(t, err)
					require.Equal(t, 1, n)
					mr.verify()

					s := FeatureString(&p[0], nil)

					assert.Equal(t, "Feature{error: geometry: panic: flatbuffers: runtime error: slice bounds out of range [773795363:8]}", s)
				}, "feature_corrupt")
			})

			t.Run("Data Section Ends before Expected Feature", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					mr := newMockDataBytesReadSeeker(t, mf, nil, nil)
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NotNil(t, hdr)
					require.NoError(t, err)
					p := make([]flat.Feature, 1)

					n, err := r.Data(p)

					assert.Same(t, io.EOF, err)
					assert.Equal(t, 0, n)
					mr.verify()
				}, "truncated_data")
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("Nil Slice", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				n, err := r.Data(nil)

				assert.Equal(t, 0, n)
				assert.NoError(t, err)
			}, seekable|notSeekable)
		})

		t.Run("Empty Slice", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				n, err := r.Data(make([]flat.Feature, 0))

				assert.Equal(t, 0, n)
				assert.NoError(t, err)
			}, seekable|notSeekable)
		})

		t.Run("No Index", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				p := make([]flat.Feature, 100)

				n, err := r.Data(p)

				assert.Greater(t, n, 0)
				assert.NoError(t, err)
			}, seekable|notSeekable, "alldatatypes.fgb", "heterogeneous.fgb", "unknown_feature_count.fgb")
		})

		t.Run("After Index", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				index, err := r.Index()
				require.NoError(t, err)
				require.NotNil(t, index)
				p := make([]flat.Feature, 100)

				n, err := r.Data(p)

				assert.Greater(t, n, 0)
				assert.NoError(t, err)
			}, seekable|notSeekable|skipNoIndex)
		})

		t.Run("Skip Index", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				p := make([]flat.Feature, 100)

				n, err := r.Data(p)

				assert.Greater(t, n, 0)
				assert.NoError(t, err)
			}, seekable|notSeekable|skipNoIndex)
		})

		t.Run("In Data Already", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				p := make([]flat.Feature, 1)
				m, err := r.Data(p)
				require.NoError(t, err)
				require.Equal(t, 1, m)

				n, err := r.Data(p)

				assert.Equal(t, n, 1)
				assert.NoError(t, err)
			}, seekable|notSeekable, "countries.fgb", "heterogeneous.fgb", "poly00.fgb", "poly01.fgb", "UScounties.fgb")
		})

		t.Run("Read All Features", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				p := make([]flat.Feature, 1)
				q := make([]flat.Feature, 1)
				var m, n int

				m, err = r.Data(p)

				assert.Equal(t, 1, m)
				assert.NoError(t, err)

				n, err = r.Data(q)

				assert.Equal(t, 0, n)
				assert.ErrorIs(t, err, io.EOF)
			}, notSeekable, "unknown_feature_count.fgb")
		})
	})
}

func TestFileReader_DataRem(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Unexpected State", func(t *testing.T) {
			t.Run("Before: Header", func(t *testing.T) {
				r := NewFileReader(&bytes.Buffer{})
				p := make([]flat.Feature, 1)

				n, err := r.Data(p)

				assert.Equal(t, 0, n)
				assert.EqualError(t, err, "flatgeobuf: must call Header()")
			})

			t.Run("Already in Error State", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					r := NewFileReader(newMockDataBytesReader(t, mf, nil))
					hdr, err1 := r.Header()
					require.EqualError(t, err1, "flatgeobuf: header length read error: EOF")
					require.Nil(t, hdr)

					p, err2 := r.DataRem()

					assert.Nil(t, p)
					assert.Same(t, err1, err2)
				}, "truncated_header")
			})
		})

		t.Run("Underlying Reader Error", func(t *testing.T) {
			t.Run("Known Feature Count", func(t *testing.T) {
				t.Run("Error Reading Feature", func(t *testing.T) {
					files := []string{"one_feature_no_index", "one_feature_with_index"}
					for _, file := range files {
						mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
							expectedErr := errors.New("problem reading the feature data")
							mr := newMockDataBytesReader(t, mf, &mockDataReadError{mf.dataOffsets[0] + 1, expectedErr})
							r := NewFileReader(mr)
							hdr, err := r.Header()
							require.NoError(t, err)
							require.NotNil(t, hdr)

							p, err1 := r.DataRem()
							err2 := r.Rewind()

							assert.NotNil(t, p)
							assert.Len(t, p, 0)
							assert.EqualError(t, err1, "flatgeobuf: feature[0] length read error (offset 0): problem reading the feature data")
							assert.ErrorIs(t, err1, expectedErr)
							assert.Same(t, err1, err2)
							mr.verify()
						}, file)
					}
				})

				t.Run("File Does Not Contain All Expected Features", func(t *testing.T) {
					mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
						r := NewFileReader(newMockDataBytesReader(t, mf, nil))
						hdr, err := r.Header()
						require.NoError(t, err)
						require.NotNil(t, hdr)
						require.Equal(t, uint64(1), hdr.FeaturesCount())

						p, err1 := r.DataRem()
						err2 := r.Rewind()

						assert.NotNil(t, p)
						assert.Len(t, p, 0)
						assert.EqualError(t, err1, "flatgeobuf: expected to read 1 features but read 0: unexpected EOF")
						assert.ErrorIs(t, err1, io.ErrUnexpectedEOF)
						assert.Same(t, err1, err2)
					}, "truncated_data")
				})
			})

			t.Run("Unknown Feature Count", func(t *testing.T) {
				t.Run("Error in First Buffer", func(t *testing.T) {
					mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
						expectedErr := errors.New("some kind of insurmountable obstacle")
						mr := newMockDataBytesReader(t, mf, &mockDataReadError{mf.dataOffsets[0] + 3, expectedErr})
						r := NewFileReader(mr)
						hdr, err := r.Header()
						require.NoError(t, err)
						require.NotNil(t, hdr)

						p, err1 := r.DataRem()
						err2 := r.Rewind()

						assert.NotNil(t, p)
						assert.Len(t, p, 0)
						assert.EqualError(t, err1, "flatgeobuf: feature[0] length read error (offset 0): some kind of insurmountable obstacle")
						assert.ErrorIs(t, err1, expectedErr)
						assert.Same(t, err1, err2)
						mr.verify()
					}, "one_feature_no_index_unknown_count")
				})

				t.Run("Error in Subsequent Buffer", func(t *testing.T) {
					mf := mockFile{
						name: "big_temporary_mock_file_with_many_features",
						header: &mockHeader{
							indexNodeSize: uint16Ptr(0),
						},
						data: make([]mockFeature, dataRemBufferSize+1),
					}
					for i := range mf.data {
						mf.data[i] = mockFeature{
							geometry: &mockGeometry{
								xy:           []float64{0, 0, float64(-i), float64(-i), float64(-i), 0, 0, 0},
								geometryType: flat.GeometryTypePolygon,
							},
						}
					}
					mf.init(t)
					expectedErr := errors.New("a stealthy problem lying in wait")
					readErr := mockDataReadError{mf.dataOffsets[dataRemBufferSize] + 13, expectedErr}
					for _, seeker := range []bool{false, true} {
						t.Run(fmt.Sprintf("Seekable: %t", seeker), func(t *testing.T) {
							var r *FileReader
							if seeker {
								mr := newMockDataBytesReadSeeker(t, &mf, &readErr, nil)
								defer mr.verify()
								r = NewFileReader(mr)
							} else {
								mr := newMockDataBytesReader(t, &mf, &readErr)
								defer mr.verify()
								r = NewFileReader(mr)
							}
							hdr, err := r.Header()
							require.NoError(t, err)
							require.NotNil(t, hdr)

							p, err1 := r.DataRem()
							err2 := r.Rewind()

							assert.NotNil(t, p)
							assert.Len(t, p, dataRemBufferSize)
							assert.EqualError(t, err1, "flatgeobuf: failed to read feature[1024] (offset=131072, len=124): a stealthy problem lying in wait")
							assert.ErrorIs(t, err1, expectedErr)
							assert.Same(t, err1, err2)
						})
					}
				})
			})
		})

		t.Run("EOF", func(t *testing.T) {
			t.Run("Empty File", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)

					p, err := r.DataRem()

					assert.NotNil(t, p)
					assert.Len(t, p, 0)
					assert.NoError(t, err)
				}, seekable|notSeekable, "empty.fgb")
			})

			t.Run("Already at EOF", func(t *testing.T) {
				t.Run("After Data", func(t *testing.T) {
					t.Run("Empty File", func(t *testing.T) {
						testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
							hdr, err := r.Header()
							require.NoError(t, err)
							require.NotNil(t, hdr)
							p := make([]flat.Feature, 1)
							n, err := r.Data(p)
							require.Equal(t, 0, n)
							require.Same(t, err, io.EOF)

							q, err := r.DataRem()

							assert.Nil(t, q)
							assert.Same(t, err, io.EOF)
						}, seekable|notSeekable, "empty.fgb")
					})

					t.Run("Known Feature Count", func(t *testing.T) {
						testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
							hdr, err := r.Header()
							require.NoError(t, err)
							require.NotNil(t, hdr)
							require.Equal(t, uint64(1), hdr.FeaturesCount())
							p := make([]flat.Feature, 1)
							n, err := r.Data(p)
							require.Equal(t, 1, n)
							require.NoError(t, err)

							q, err := r.DataRem()

							assert.Nil(t, q)
							assert.Same(t, err, io.EOF)
						}, seekable|notSeekable, "alldatatypes.fgb")
					})

					t.Run("Unknown Feature Count", func(t *testing.T) {
						t.Run("Data Does Not Reach EOF State", func(t *testing.T) {
							testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
								hdr, err := r.Header()
								require.NoError(t, err)
								require.NotNil(t, hdr)
								require.Equal(t, uint64(0), hdr.FeaturesCount())
								p := make([]flat.Feature, 1)
								n, err := r.Data(p)
								require.Equal(t, 1, n)
								require.NoError(t, err)

								q, err := r.DataRem()

								assert.NotNil(t, q)
								assert.Len(t, q, 0)
								assert.NoError(t, err)
							}, seekable|notSeekable, "unknown_feature_count.fgb")
						})

						t.Run("Data Reaches EOF State", func(t *testing.T) {
							testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
								hdr, err := r.Header()
								require.NoError(t, err)
								require.NotNil(t, hdr)
								require.Equal(t, uint64(0), hdr.FeaturesCount())
								p := make([]flat.Feature, 2)
								n, err := r.Data(p)
								require.Equal(t, 1, n)
								require.NoError(t, err)

								q, err := r.DataRem()

								assert.Nil(t, q)
								assert.Same(t, err, io.EOF)
							}, seekable|notSeekable, "unknown_feature_count.fgb")
						})
					})
				})

				t.Run("After DataRem", func(t *testing.T) {
					testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
						hdr, err := r.Header()
						require.NoError(t, err)
						require.NotNil(t, hdr)
						p, err := r.DataRem()
						assert.NotNil(t, p)
						assert.NoError(t, err)

						q, err := r.DataRem()

						assert.Nil(t, q)
						assert.Same(t, err, io.EOF)
					}, seekable|notSeekable)
				})
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("No Index", func(t *testing.T) {
			mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
				r := NewFileReader(newMockDataBytesReader(t, mf, nil))
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				p, err := r.DataRem()

				assert.NoError(t, err)
				require.Len(t, p, 1)
				g := p[0].Geometry(nil)
				require.NotNil(t, g)
				assert.Equal(t, mf.data[0].geometry.geometryType, g.Type())
				require.Equal(t, len(mf.data[0].geometry.xy), g.XyLength())
				for i, expected := range mf.data[0].geometry.xy {
					actual := g.Xy(i)
					assert.Equal(t, expected, actual, "coordinate mismatch at xy[%d]", i)
				}
			}, "one_feature_no_index")
		})

		t.Run("After Index", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				_, _ = r.Index()

				p, err := r.DataRem()

				assert.NotNil(t, p)
				assert.NoError(t, err)
			}, seekable|notSeekable)
		})

		t.Run("Skip Index", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				p, err := r.DataRem()

				assert.NotNil(t, p)
				assert.NoError(t, err)
			}, seekable|notSeekable)
		})

		t.Run("In Data Already", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)
				p := make([]flat.Feature, 1)
				n, err := r.Data(p)
				require.Equal(t, 1, n)
				require.NoError(t, err)

				q, err := r.DataRem()

				assert.NotNil(t, q)
				assert.NoError(t, err)
			}, seekable|notSeekable, "countries.fgb", "heterogeneous.fgb", "poly00.fgb", "poly01.fgb", "UScounties.fgb")
		})
	})
}

func TestFileReader_Rewind(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		t.Run("Unexpected State", func(t *testing.T) {
			t.Run("Header Not Called", func(t *testing.T) {
				r := NewFileReader(&bytes.Buffer{})

				err := r.Rewind()

				assert.EqualError(t, err, "flatgeobuf: must call Header()")
			})

			t.Run("Already in Error State", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					r := NewFileReader(newMockDataBytesReader(t, mf, nil))
					hdr, err1 := r.Header()
					require.EqualError(t, err1, "flatgeobuf: header length read error: EOF")
					require.Nil(t, hdr)

					err2 := r.Rewind()

					assert.Same(t, err1, err2)
				}, "truncated_header")
			})
		})

		t.Run("Usage Error", func(t *testing.T) {
			t.Run("Not Seekable", func(t *testing.T) {
				r := newTestDataFileReader(t, false, "empty.fgb")
				_, err := r.Header()
				require.NoError(t, err)

				err = r.Rewind()

				assert.EqualError(t, err, "flatgeobuf: can't rewind: reader is not an io.Seeker")
			})
		})

		t.Run("Underlying Reader Error", func(t *testing.T) {
			t.Run("Failed to Seek to End of Header", func(t *testing.T) {
				mockDataRunTest(t, func(t *testing.T, mf *mockFile) {
					expectedErr := errors.New("bam")
					mr := newMockDataBytesReadSeeker(t, mf, nil, &mockDataSeekError{mf.indexOffset, io.SeekStart, 0, expectedErr})
					r := NewFileReader(mr)
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					index, err := r.Index()
					require.NoError(t, err)
					require.NotNil(t, index)

					err = r.Rewind()

					assert.EqualError(t, err, "flatgeobuf: failed to seek to end of header: bam")
					assert.ErrorIs(t, err, expectedErr)
					mr.verify()
				}, "one_feature_with_index")
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("State: After Header", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
				hdr, err := r.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr)

				err = r.Rewind()

				assert.NoError(t, err)
			}, seekable)
		})

		t.Run("State: After Index", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
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
			}, seekable)
		})

		t.Run("State: In Data", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
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
			}, seekable)
		})

		t.Run("State: EOF", func(t *testing.T) {
			t.Run("After DataRem", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
					hdr, err := r.Header()
					require.NoError(t, err)
					require.NotNil(t, hdr)
					p, err1 := r.DataRem()
					if p != nil || !errors.Is(err, io.EOF) {
						require.NoError(t, err)
					}

					err = r.Rewind()
					assert.NoError(t, err)

					q, err2 := r.DataRem()
					assert.Equal(t, err1, err2)
					assert.Equal(t, p, q)
				}, seekable)
			})

			t.Run("After IndexSearch", func(t *testing.T) {
				testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
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
				}, seekable)
			})
		})

		t.Run("Repeated", func(t *testing.T) {
			testDataRunTests(t, func(t *testing.T, r *FileReader, _ string) {
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
							p, err := r.DataRem()
							if p != nil || !errors.Is(err, io.EOF) {
								require.NoError(t, err)
							}
						}

						err := r.Rewind()
						assert.NoError(t, err)
					})
				}
			}, seekable)
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
