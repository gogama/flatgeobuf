// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gogama/flatgeobuf/packedrtree"
	"io"
	"math"
	"sort"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"

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
		t.Run("Nil Header", func(t *testing.T) {
			w := NewFileWriter(&bytes.Buffer{})

			assert.PanicsWithValue(t, "flatgeobuf: nil header", func() {
				_, _ = w.Header(nil)
			})
		})

		t.Run("Corrupt Header: Failed to Get Feature Count", func(t *testing.T) {
			bldr := flatbuffers.NewBuilder(0)
			flat.HeaderStart(bldr)
			flat.HeaderAddFeaturesCount(bldr, 101)
			offset := flat.HeaderEnd(bldr)
			flat.FinishHeaderBuffer(bldr, offset)
			b := bldr.FinishedBytes()
			b[27] = 0xff // Corrupt the header's features_count field.
			hdr := flat.GetRootAsHeader(b, 0)
			w := NewFileWriter(&bytes.Buffer{})

			n, err := w.Header(hdr)

			assert.EqualError(t, err, "flatgeobuf: failed to get header feature count: panic: flatbuffers: runtime error: slice bounds out of range [65312:40]")
			assert.Equal(t, 0, n)
		})

		t.Run("Corrupt Header: Failed to Get Index Node Size", func(t *testing.T) {
			bldr := flatbuffers.NewBuilder(0)
			flat.HeaderStart(bldr)
			flat.HeaderAddFeaturesCount(bldr, 155)
			flat.HeaderAddIndexNodeSize(bldr, 156)
			offset := flat.HeaderEnd(bldr)
			flat.FinishHeaderBuffer(bldr, offset)
			b := bldr.FinishedBytes()
			b[30] = 0xff // Corrupt the header's index_node_size field.
			hdr := flat.GetRootAsHeader(b, 0)
			w := NewFileWriter(&bytes.Buffer{})

			n, err := w.Header(hdr)

			assert.EqualError(t, err, "flatgeobuf: failed to get header index node size: panic: flatbuffers: runtime error: slice bounds out of range [287:48]")
			assert.Equal(t, 0, n)
		})

		t.Run("Illegal Index Node Size", func(t *testing.T) {
			bad := mockHeader{indexNodeSize: uint16Ptr(1)}
			good := mockHeader{}
			w := NewFileWriter(&bytes.Buffer{})

			n, err := w.Header(bad.buildAsTable())

			assert.EqualError(t, err, "flatgeobuf: index node size may not be 1")
			assert.Equal(t, 0, n)

			n, err = w.Header(good.buildAsTable())

			assert.NoError(t, err)
			assert.Greater(t, n, 0)
		})

		t.Run("Header Already Called", func(t *testing.T) {
			w := NewFileWriter(&bytes.Buffer{})
			hdr := (&mockHeader{}).buildAsTable()
			n, err := w.Header(hdr)
			require.NoError(t, err)
			require.Greater(t, n, 0)

			n, err = w.Header(hdr)

			require.EqualError(t, err, "flatgeobuf: Header() has already been called")
			require.Equal(t, 0, n)
		})

		t.Run("Already in Error State", func(t *testing.T) {
			expectedErr := errors.New("foo")
			m := newMockWriteCloser(t)
			m.
				On("Write", mock.Anything).
				Return(0, expectedErr).
				Once()
			m.
				On("Write", mock.Anything).
				Return(-1, nil).
				Times(0)
			w := NewFileWriter(m)
			hdr := (&mockHeader{}).buildAsTable()
			n, err := w.Header(hdr)
			require.EqualError(t, err, "flatgeobuf: failed to write magic number: foo")
			require.Equal(t, 0, n)

			n, err = w.Header(hdr)

			require.EqualError(t, err, "flatgeobuf: failed to write magic number: foo")
			require.Equal(t, 0, n)
			m.AssertExpectations(t)
		})

		t.Run("Table Has Invalid Size Prefix", func(t *testing.T) {
			// Create a valid size-prefixed header.
			bldr := flatbuffers.NewBuilder(0)
			flat.HeaderStart(bldr)
			flat.HeaderAddFeaturesCount(bldr, 0)
			offset := flat.HeaderEnd(bldr)
			flat.FinishSizePrefixedFeatureBuffer(bldr, offset)
			b := bldr.FinishedBytes()
			hdr := flat.GetSizePrefixedRootAsHeader(b, 0)
			// Corrupt the buffer by setting the size prefix to an
			// exorbitantly high value.
			flatbuffers.WriteUint32(b, math.MaxUint32)
			// Create file writer under test.
			w := NewFileWriter(&bytes.Buffer{})

			n, err := w.Header(hdr)

			assert.EqualError(t, err, "flatgeobuf: failed to write header: flatgeobuf: FlatBuffers table buffer is smaller than size prefix indicates (need=4+4294967295, len=16, gap=4294967283)")
			assert.Equal(t, magicLen, n)
		})

		t.Run("Underlying Writer Error", func(t *testing.T) {
			expectedErr := errors.New("bar")
			m := newMockWriteCloser(t)
			m.
				On("Write", magic[:]).
				Return(magicLen, nil).
				Once()
			m.
				On("Write", mock.Anything).
				Return(17, expectedErr).
				Once()
			w := NewFileWriter(m)
			hdr := (&mockHeader{}).buildAsTable()

			n, err := w.Header(hdr)

			assert.EqualError(t, err, "flatgeobuf: failed to write header: bar")
			assert.Equal(t, n, magicLen+17)
			m.AssertExpectations(t)
		})
	})

	t.Run("Success", func(t *testing.T) {
		testCases := []struct {
			name string
			mh   mockHeader
		}{
			{
				name: "FeaturesCount=0",
				mh: mockHeader{
					featuresCount: 0,
					indexNodeSize: uint16Ptr(16), // Needed because of FBS default
				},
			},
			{
				name: "IndexNodeSize=2",
				mh: mockHeader{
					indexNodeSize: uint16Ptr(2),
				},
			},
			{
				name: "Named",
				mh: mockHeader{
					name:          stringPtr("hello, world"),
					indexNodeSize: uint16Ptr(16), // Needed because the FBS defaults to 16 if omitted
				},
			},
			{
				name: "WithSchema",
				mh: mockHeader{
					indexNodeSize: uint16Ptr(16), // Needed because the FBS defaults to 16 if omitted
					columns: []mockColumn{
						{
							name:       "bin",
							columnType: flat.ColumnTypeBinary,
							width:      1,    // Needed because the FBS defaults to -1 if omitted
							precision:  2,    // Needed because the FBS defaults to -1 if omitted
							scale:      3,    // Needed because the FBS defaults to -1 if omitted
							nullable:   true, // Needed because the FBS defaults to true if omitted
						},
						{
							name:       "bool",
							columnType: flat.ColumnTypeBool,
							width:      4,     // Needed because the FBS defaults to -1 if omitted
							precision:  5,     // Needed because the FBS defaults to -1 if omitted
							scale:      6,     // Needed because the FBS defaults to -1 if omitted
							nullable:   false, // Needed because the FBS defaults to true if omitted
						},
					},
				},
			},
			{
				name: "Everything",
				mh: mockHeader{
					name:         stringPtr("everything"),
					envelope:     []float64{-2, -3, 4, 5},
					geometryType: flat.GeometryTypeGeometryCollection,
					hasZ:         true,
					hasM:         true,
					hasT:         true,
					hasTM:        true,
					columns: []mockColumn{
						{
							name:       "bin",
							columnType: flat.ColumnTypeBinary,
							width:      1,    // Needed because the FBS defaults to -1 if omitted
							precision:  2,    // Needed because the FBS defaults to -1 if omitted
							scale:      3,    // Needed because the FBS defaults to -1 if omitted
							nullable:   true, // Needed because the FBS defaults to true if omitted
						},
					},
					featuresCount: 111,
					indexNodeSize: uint16Ptr(222),
					crs: &mockCRS{
						org:         stringPtr("gogama"),
						code:        7,
						name:        stringPtr("flatgeobuf"),
						description: stringPtr("Native Go library implementing FlatGeobuf, a performant binary encoding for geographic data based on FlatBuffers."),
						wkt:         stringPtr("ham"),
						codeString:  stringPtr("eggs"),
					},
					title:       stringPtr("my file"),
					description: stringPtr("it contains my data"),
					metadata:    stringPtr("and this is my metadata!"),
				},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				var b bytes.Buffer
				tableBytes := testCase.mh.buildAsBytes()
				inHdr := flat.GetSizePrefixedRootAsHeader(tableBytes, 0)

				t.Run("Write", func(t *testing.T) {
					w := NewFileWriter(&b)

					n, err := w.Header(inHdr)

					assert.NoError(t, err)
					assert.Equal(t, magicLen+len(tableBytes), n)
				})

				t.Run("Verify", func(t *testing.T) {
					r := NewFileReader(&b)

					outHdr, err := r.Header()

					assert.NoError(t, err)
					assert.Equal(t, mockHeaderFromFlatBufferTable(outHdr), testCase.mh)
				})
			})
		}
	})
}

func TestFileWriter_Index(t *testing.T) {
	const numRefs = 1
	const nodeSize = 2
	index, _ := packedrtree.New(make([]packedrtree.Ref, numRefs), nodeSize)
	require.NotNil(t, index)

	t.Run("Error", func(t *testing.T) {
		t.Run("Nil Index", func(t *testing.T) {
			w := NewFileWriter(&bytes.Buffer{})

			assert.PanicsWithValue(t, "flatgeobuf: nil index", func() {
				_, _ = w.Index(nil)
			})
		})

		t.Run("Can't Write Index", func(t *testing.T) {
			t.Run("Already in Error State", func(t *testing.T) {
				expectedErr := errors.New("foo")
				n := 1
				m := newMockWriteCloser(t)
				m.
					On("Write", mock.Anything).
					Return(n, expectedErr).
					Once()
				w := NewFileWriter(m)
				o, err := w.Header((&mockHeader{}).buildAsTable())
				require.ErrorIs(t, err, expectedErr)
				require.Equal(t, n, o)

				o, err = w.Index(index)

				assert.ErrorIs(t, err, expectedErr)
				assert.Equal(t, 0, o)
				m.AssertExpectations(t)
			})

			t.Run("Header Not Called", func(t *testing.T) {
				w := NewFileWriter(&bytes.Buffer{})

				n, err := w.Index(index)

				assert.EqualError(t, err, "flatgeobuf: must call Header()")
				assert.Equal(t, 0, n)
			})

			t.Run("Index Node Size is Zero", func(t *testing.T) {
				mh := mockHeader{
					indexNodeSize: uint16Ptr(0),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Index(index)

				assert.EqualError(t, err, "flatgeobuf: header node size 0 indicates no index")
				assert.Equal(t, 0, n)
			})

			t.Run("Already Wrote Index", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: numRefs,
					indexNodeSize: uint16Ptr(nodeSize),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)
				n, err = w.Index(index)
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Index(index)

				assert.EqualError(t, err, "flatgeobuf: write position is past index")
				assert.Equal(t, 0, n)
			})

			t.Run("Already Wrote Some Data", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: 2,
					indexNodeSize: uint16Ptr(0),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)
				mf := mockFeature{
					geometry: &mockGeometry{
						xy:           []float64{0, 0},
						geometryType: flat.GeometryTypePoint,
					},
				}
				n, err = w.Data([]flat.Feature{*mf.buildAsTable()})
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Index(index)

				assert.EqualError(t, err, "flatgeobuf: write position is past index")
				assert.Equal(t, 0, n)
			})
		})

		t.Run("Index Write Error", func(t *testing.T) {
			t.Run("Feature Count Mismatch", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: numRefs + 1,
					indexNodeSize: uint16Ptr(nodeSize),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Index(index)

				assert.EqualError(t, err, "flatgeobuf: feature count mismatch (header=2, index=1)")
				assert.Equal(t, 0, n)
			})

			t.Run("Index Node Size Mismatch", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: numRefs,
					indexNodeSize: uint16Ptr(nodeSize + 1),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Index(index)

				assert.EqualError(t, err, "flatgeobuf: node size mismatch (header=3, index=2)")
				assert.Equal(t, 0, n)
			})

			t.Run("Underlying Writer Error", func(t *testing.T) {
				m := newMockWriteCloser(t)
				m.
					On("Write", mock.MatchedBy(func(p []byte) bool {
						return bytes.Equal(p, magic[:])
					})).
					Return(magicLen, nil).
					Once()
				headerWriteCall := m.
					On("Write", mock.Anything).
					Once()
				headerWriteCall.RunFn = func(args mock.Arguments) {
					p := args[0].([]byte)
					headerWriteCall.ReturnArguments = mock.Arguments{len(p), nil}
				}
				expectedErr := errors.New("magnetic tape snarl")
				m.
					On("Write", mock.Anything).
					Return(0, expectedErr).
					Once()
				w := NewFileWriter(m)
				n, err := w.Header((&mockHeader{
					featuresCount: numRefs,
					indexNodeSize: uint16Ptr(nodeSize),
				}).buildAsTable())
				require.NoError(t, err)
				require.Equal(t, magicLen+headerWriteCall.ReturnArguments[0].(int), n)

				n, err = w.Index(index)

				assert.EqualError(t, err, "flatgeobuf: failed to write index: magnetic tape snarl")
				assert.ErrorIs(t, err, expectedErr)
				assert.Equal(t, 0, n)
				m.AssertExpectations(t)
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		w := NewFileWriter(&bytes.Buffer{})
		m, err := packedrtree.Size(numRefs, nodeSize)
		require.NoError(t, err)
		require.Greater(t, m, 0)
		n, err := w.Header((&mockHeader{
			featuresCount: numRefs,
			indexNodeSize: uint16Ptr(nodeSize),
		}).buildAsTable())
		require.NoError(t, err)
		require.Greater(t, n, 0)

		n, err = w.Index(index)

		assert.NoError(t, err)
		assert.Equal(t, m, n)
	})
}

func TestFileWriter_IndexData(t *testing.T) {
	const numRefs = 1
	const nodeSize = 2
	origin := mockFeature{
		geometry: &mockGeometry{
			xy:           []float64{0, 0},
			geometryType: flat.GeometryTypePoint,
		},
	}

	t.Run("Error", func(t *testing.T) {
		t.Run("Can't Write Index", func(t *testing.T) {
			t.Run("Already in Error State", func(t *testing.T) {
				expectedErr := errors.New("foo")
				n := 1
				m := newMockWriteCloser(t)
				m.
					On("Write", mock.Anything).
					Return(n, expectedErr).
					Once()
				w := NewFileWriter(m)
				o, err := w.Header((&mockHeader{}).buildAsTable())
				require.ErrorIs(t, err, expectedErr)
				require.Equal(t, n, o)

				o, err = w.IndexData(nil)

				assert.ErrorIs(t, err, expectedErr)
				assert.Equal(t, 0, o)
				m.AssertExpectations(t)
			})

			t.Run("Header Not Called", func(t *testing.T) {
				w := NewFileWriter(&bytes.Buffer{})

				n, err := w.IndexData(nil)

				assert.EqualError(t, err, "flatgeobuf: must call Header()")
				assert.Equal(t, 0, n)
			})

			t.Run("Index Node Size is Zero", func(t *testing.T) {
				mh := mockHeader{
					indexNodeSize: uint16Ptr(0),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.IndexData(nil)

				assert.EqualError(t, err, "flatgeobuf: header node size 0 indicates no index")
				assert.Equal(t, 0, n)
			})

			t.Run("Already Wrote Index", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: numRefs,
					indexNodeSize: uint16Ptr(nodeSize),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)
				n, err = w.IndexData([]flat.Feature{*origin.buildAsTable()})
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.IndexData(nil)

				assert.EqualError(t, err, "flatgeobuf: write position is past index")
				assert.Equal(t, 0, n)
			})

			t.Run("Already Wrote Some Data", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: 2,
					indexNodeSize: uint16Ptr(0),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)
				n, err = w.Data([]flat.Feature{*origin.buildAsTable()})
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.IndexData([]flat.Feature{*origin.buildAsTable()})

				assert.EqualError(t, err, "flatgeobuf: write position is past index")
				assert.Equal(t, 0, n)
			})
		})

		t.Run("Index Write Error", func(t *testing.T) {
			t.Run("Corrupt Feature: Table Size", func(t *testing.T) {
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header((&mockHeader{
					featuresCount: 1,
					indexNodeSize: uint16Ptr(nodeSize),
				}).buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.IndexData([]flat.Feature{{}})

				assert.EqualError(t, err, "flatgeobuf: failed to index feature 0: flatgeobuf: feature 0: flatgeobuf: FlatBuffers table buffer is too small for a size prefix (need=4, len=0)")
				assert.Equal(t, 0, n)
			})

			t.Run("Corrupt Feature: Bounds", func(t *testing.T) {
				bldr := flatbuffers.NewBuilder(0)
				geometryOffset := bldr.StartVector(flatbuffers.SizeFloat64, 2, flatbuffers.SizeFloat64)
				bldr.PrependFloat64(-2)
				bldr.PrependFloat64(-3)
				bldr.EndVector(2)
				flat.FeatureStart(bldr)
				flat.FeatureAddGeometry(bldr, geometryOffset)
				offset := flat.FeatureEnd(bldr)
				flat.FinishFeatureBuffer(bldr, offset)
				b := bldr.FinishedBytes()
				b[11] = 0xff // Corrupt the feature's geometry field
				f := flat.GetRootAsFeature(b, 0)
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header((&mockHeader{
					featuresCount: 1,
					indexNodeSize: uint16Ptr(nodeSize),
				}).buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.IndexData([]flat.Feature{*f})

				assert.EqualError(t, err, "flatgeobuf: failed to index feature 0: flatgeobuf: feature 0: panic: flatbuffers: runtime error: slice bounds out of range [16777220:32]")
				assert.Equal(t, 0, n)
			})

			t.Run("Feature Count Mismatch", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: numRefs + 1,
					indexNodeSize: uint16Ptr(nodeSize),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.IndexData([]flat.Feature{*origin.buildAsTable()})

				assert.EqualError(t, err, "flatgeobuf: feature count mismatch (header=2, index=1)")
				assert.Equal(t, 0, n)
			})

			t.Run("Underlying Writer Error", func(t *testing.T) {
				m := newMockWriteCloser(t)
				m.
					On("Write", mock.MatchedBy(func(p []byte) bool {
						return bytes.Equal(p, magic[:])
					})).
					Return(magicLen, nil).
					Once()
				headerWriteCall := m.
					On("Write", mock.Anything).
					Once()
				headerWriteCall.RunFn = func(args mock.Arguments) {
					p := args[0].([]byte)
					headerWriteCall.ReturnArguments = mock.Arguments{len(p), nil}
				}
				expectedErr := errors.New("magnetic tape snarl")
				m.
					On("Write", mock.Anything).
					Return(0, expectedErr).
					Once()
				w := NewFileWriter(m)
				n, err := w.Header((&mockHeader{
					featuresCount: numRefs,
					indexNodeSize: uint16Ptr(nodeSize),
				}).buildAsTable())
				require.NoError(t, err)
				require.Equal(t, magicLen+headerWriteCall.ReturnArguments[0].(int), n)

				n, err = w.IndexData([]flat.Feature{*origin.buildAsTable()})

				assert.EqualError(t, err, "flatgeobuf: failed to write index: magnetic tape snarl")
				assert.ErrorIs(t, err, expectedErr)
				assert.Equal(t, 0, n)
				m.AssertExpectations(t)
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		var b bytes.Buffer
		mh := mockHeader{
			featuresCount: 2,
			indexNodeSize: uint16Ptr(3),
		}
		extraFeature := mockFeature{
			geometry: &mockGeometry{
				ends:         []uint32{9},
				xy:           []float64{0, 0, 0, 1, 1, 1, 1, 0, 0, 0},
				geometryType: flat.GeometryTypePolygon,
			},
		}
		var index *packedrtree.PackedRTree

		t.Run("Write", func(t *testing.T) {
			w := NewFileWriter(&b)
			n, err := w.Header(mh.buildAsTable())
			require.NoError(t, err)
			require.Greater(t, n, 0)

			n, err = w.IndexData([]flat.Feature{*origin.buildAsTable(), *extraFeature.buildAsTable()})

			assert.NoError(t, err)
			assert.Greater(t, n, 0)

		})

		t.Run("Read", func(t *testing.T) {
			r := NewFileReader(&b)
			hdr, err := r.Header()
			require.NoError(t, err)
			require.NotNil(t, hdr)
			require.Equal(t, mh, mockHeaderFromFlatBufferTable(hdr))

			index, err = r.Index()

			assert.NoError(t, err)
			require.NotNil(t, index)
			assert.Equal(t, 2, index.NumRefs())
			assert.Equal(t, uint16(3), index.NodeSize())
		})

		t.Run("Search", func(t *testing.T) {
			assert.Equal(t, packedrtree.Results{}, index.Search(packedrtree.EmptyBox))

			sr := index.Search(packedrtree.Box{})
			sort.Sort(sr)

			assert.Equal(t, packedrtree.Results{
				{
					Offset:   0,
					RefIndex: 1,
				},
				{
					Offset:   80,
					RefIndex: 0,
				},
			}, sr)
		})
	})
}

func TestFileWriter_Data(t *testing.T) {
	origin := mockFeature{
		geometry: &mockGeometry{
			xy:           []float64{0, 0},
			geometryType: flat.GeometryTypePoint,
		},
	}

	t.Run("Error", func(t *testing.T) {
		t.Run("Can't Write Data", func(t *testing.T) {
			t.Run("Already in Error State", func(t *testing.T) {
				expectedErr := errors.New("bar")
				n := 1
				m := newMockWriteCloser(t)
				m.
					On("Write", mock.Anything).
					Return(n, expectedErr).
					Once()
				w := NewFileWriter(m)
				o, err := w.Header((&mockHeader{}).buildAsTable())
				require.ErrorIs(t, err, expectedErr)
				require.Equal(t, n, o)

				o, err = w.Data(nil)

				assert.ErrorIs(t, err, expectedErr)
				assert.Equal(t, 0, o)
				m.AssertExpectations(t)
			})

			t.Run("Header Not Called", func(t *testing.T) {
				w := NewFileWriter(&bytes.Buffer{})

				n, err := w.Data(nil)

				assert.EqualError(t, err, "flatgeobuf: must call Header()")
				assert.Equal(t, 0, n)
			})

			t.Run("Index Not Written", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: 1,
					indexNodeSize: uint16Ptr(2),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Data(nil)

				assert.EqualError(t, err, "flatgeobuf: header specifies index but no index written")
				assert.Equal(t, 0, n)
			})

			t.Run("All Features Already Written", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: 1,
					indexNodeSize: uint16Ptr(0),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)
				n, err = w.Data([]flat.Feature{*origin.buildAsTable()})
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Data([]flat.Feature{*origin.buildAsTable()})

				assert.EqualError(t, err, "flatgeobuf: all 1 features indicated in header already written")
				assert.Equal(t, 0, n)
			})

			t.Run("Excess Features Requested", func(t *testing.T) {
				mh := mockHeader{
					featuresCount: 1,
					indexNodeSize: uint16Ptr(0),
				}
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header(mh.buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Data([]flat.Feature{*origin.buildAsTable(), *origin.buildAsTable()})

				assert.EqualError(t, err, "flatgeobuf: 0 of 1 features indicated in header already written, writing 2 more would create an excess of 1")
				assert.Equal(t, 0, n)
			})
		})

		t.Run("Data Write Error", func(t *testing.T) {
			t.Run("Corrupt Feature: Table Size", func(t *testing.T) {
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header((&mockHeader{
					featuresCount: 1,
					indexNodeSize: uint16Ptr(0),
				}).buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Data([]flat.Feature{{}})

				assert.EqualError(t, err, "flatgeobuf: failed to write feature 0 at data index 0: flatgeobuf: FlatBuffers table buffer is too small for a size prefix (need=4, len=0)")
				assert.Equal(t, 0, n)
			})

			t.Run("Corrupt Feature: Buffer too Small", func(t *testing.T) {
				b := make([]byte, flatbuffers.SizeUint32)
				f := flat.GetRootAsFeature(b, 0)
				flatbuffers.WriteUint32(b, 100001)
				w := NewFileWriter(&bytes.Buffer{})
				n, err := w.Header((&mockHeader{
					featuresCount: 1,
					indexNodeSize: uint16Ptr(0),
				}).buildAsTable())
				require.NoError(t, err)
				require.Greater(t, n, 0)

				n, err = w.Data([]flat.Feature{*f})

				assert.EqualError(t, err, "flatgeobuf: failed to write feature 0 at data index 0: flatgeobuf: FlatBuffers table buffer is smaller than size prefix indicates (need=4+100001, len=4, gap=100001)")
				assert.Equal(t, 0, n)
			})

			t.Run("Underlying Writer Error", func(t *testing.T) {
				m := newMockWriteCloser(t)
				m.
					On("Write", mock.MatchedBy(func(p []byte) bool {
						return bytes.Equal(p, magic[:])
					})).
					Return(magicLen, nil).
					Once()
				headerWriteCall := m.
					On("Write", mock.Anything).
					Once()
				headerWriteCall.RunFn = func(args mock.Arguments) {
					p := args[0].([]byte)
					headerWriteCall.ReturnArguments = mock.Arguments{len(p), nil}
				}
				expectedErr := errors.New("output device is napping")
				m.
					On("Write", mock.Anything).
					Return(0, expectedErr).
					Once()
				w := NewFileWriter(m)
				n, err := w.Header((&mockHeader{
					featuresCount: 10,
					indexNodeSize: uint16Ptr(0),
				}).buildAsTable())
				require.NoError(t, err)
				require.Equal(t, magicLen+headerWriteCall.ReturnArguments[0].(int), n)

				n, err = w.Data([]flat.Feature{*origin.buildAsTable()})

				assert.EqualError(t, err, "flatgeobuf: failed to write feature 0 at data index 0: output device is napping")
				assert.ErrorIs(t, err, expectedErr)
				assert.Equal(t, 0, n)
				m.AssertExpectations(t)
			})
		})
	})

	t.Run("Success", func(t *testing.T) {
		t.Run("Simple", func(t *testing.T) {
			testCases := []struct {
				name     string
				features []mockFeature
			}{
				{
					name: "Nil",
				},
				{
					name:     "Empty",
					features: make([]mockFeature, 0),
				},
				{
					name:     "One",
					features: []mockFeature{origin},
				},
				{
					name: "Two",
					features: []mockFeature{
						origin,
						{
							geometry: &mockGeometry{
								geometryType: flat.GeometryTypeGeometryCollection,
								parts: []mockGeometry{
									{
										xy:           []float64{-2, -2, -1, -1},
										geometryType: flat.GeometryTypeLineString,
									},
									{
										ends:         []uint32{3, 7},
										xy:           []float64{1, 1, 2, 2, 3, 3, 4, 4},
										geometryType: flat.GeometryTypeMultiLineString,
									},
								},
							},
						},
					},
				},
			}
			for _, testCase := range testCases {
				t.Run(testCase.name, func(t *testing.T) {
					for i, indexed := range []string{"No Index", "Index"} {
						t.Run(indexed, func(t *testing.T) {
							if i == 1 && len(testCase.features) == 0 {
								t.Skip("Can't index empty file")
							}
							var b bytes.Buffer
							mh := mockHeader{
								featuresCount: uint64(len(testCase.features)),
								indexNodeSize: uint16Ptr(uint16(i * 2)),
							}

							t.Run("Write", func(t *testing.T) {
								w := NewFileWriter(&b)
								n, err := w.Header(mh.buildAsTable())
								require.NoError(t, err)
								require.Greater(t, n, 0)
								var p []flat.Feature
								if testCase.features != nil {
									p = make([]flat.Feature, len(testCase.features))
									for j := range testCase.features {
										p[j] = *testCase.features[j].buildAsTable()
									}
								}
								if i == 0 {
									n, err = w.Data(p)
								} else {
									n, err = w.IndexData(p)
								}

								assert.NoError(t, err)
								if len(testCase.features) == 0 {
									assert.Equal(t, 0, n)
								} else {
									assert.Greater(t, n, 0)
								}
							})

							if t.Failed() {
								return
							}

							r := NewFileReader(bytes.NewReader(b.Bytes()))

							t.Run("Read", func(t *testing.T) {
								hdr, err := r.Header()
								require.NoError(t, err)
								require.Equal(t, mh.buildAsTable(), hdr)

								p, err := r.DataRem()

								require.Len(t, p, len(testCase.features))
								if testCase.features != nil {
									features := make([]mockFeature, len(testCase.features))
									for j := range features {
										features[j] = mockFeatureFromFlatBufferTable(&p[j])
									}
									assert.Equal(t, testCase.features, features)
								}
							})

							t.Run("Search", func(t *testing.T) {
								if i == 0 || len(testCase.features) == 0 {
									t.Skip("Can't search without index")
								}
								err := r.Rewind()
								require.NoError(t, err)

								p, err := r.IndexSearch(packedrtree.Box{})

								assert.Len(t, p, 1)
							})
						})
					}
				})
			}
		})

		t.Run("Repeat Calls", func(t *testing.T) {
			columns := []mockColumn{
				{
					name:       "foo_column",
					columnType: flat.ColumnTypeString,
					width:      -1,
					precision:  -1,
					scale:      -1,
					nullable:   false,
					primaryKey: true,
					unique:     true,
				},
				{
					name:        "bar_column",
					columnType:  flat.ColumnTypeShort,
					description: stringPtr("baz description"),
					width:       -1,
					precision:   -1,
					scale:       -1,
				},
			}
			makeProps := func(foo string, bar int16) []byte {
				var b bytes.Buffer
				var err error
				w := NewPropWriter(&b)
				_, err = w.WriteString(foo)
				require.NoError(t, err)
				if bar != 0 {
					_, err = w.WriteShort(bar)
					require.NoError(t, err)
				}
				return b.Bytes()
			}
			expected := [][]mockFeature{
				{
					origin,
				},
				{
					{
						geometry: &mockGeometry{
							xy:           []float64{-1, -1},
							geometryType: flat.GeometryTypePoint,
							z:            []float64{-1},
						},
						properties: makeProps("hello", -5),
						columns:    columns,
					},
					{
						geometry: &mockGeometry{
							xy:           []float64{-10, 10, -8, 10, -8, 8, -10, 8, -10, 10},
							geometryType: flat.GeometryTypePolygon,
						},
					},
				},
				{
					{
						geometry: &mockGeometry{
							xy:           []float64{1, 1, 2, 2},
							geometryType: flat.GeometryTypeLineString,
						},
						properties: makeProps("world", 0),
						columns:    columns,
					},
				},
			}
			var numRefs uint64
			for i := range expected {
				numRefs += uint64(len(expected[i]))
			}

			for _, knownFeatureCount := range []bool{false, true} {
				t.Run(fmt.Sprintf("Known Feature Count: %t", knownFeatureCount), func(t *testing.T) {
					var b bytes.Buffer
					mh := mockHeader{
						indexNodeSize: uint16Ptr(0),
					}
					if knownFeatureCount {
						mh.featuresCount = numRefs
					}

					t.Run("Write", func(t *testing.T) {
						w := NewFileWriter(&b)
						n, err := w.Header(mh.buildAsTable())
						require.NoError(t, err)
						require.Greater(t, n, 0)

						for i := range expected {
							t.Run(fmt.Sprintf("Expected[%d]", i), func(t *testing.T) {
								p := make([]flat.Feature, len(expected[i]))
								for j := range expected[i] {
									p[j] = *expected[i][j].buildAsTable()
								}
								_, err = w.Data(p)
								require.NoError(t, err)
							})
						}
					})

					if t.Failed() {
						return
					}

					t.Run("Read", func(t *testing.T) {
						r := NewFileReader(&b)
						hdr, err := r.Header()
						require.NoError(t, err)
						require.Equal(t, mh, mockHeaderFromFlatBufferTable(hdr))

						for i := range expected {
							t.Run(fmt.Sprintf("Expected[%d]", i), func(t *testing.T) {
								p := make([]flat.Feature, len(expected[i]))
								n, err := r.Data(p)
								require.NoError(t, err)
								require.Equal(t, len(expected[i]), n)
								for j := range expected[i] {
									actual := mockFeatureFromFlatBufferTable(&p[j])
									assert.Equal(t, expected[i][j], actual, "Feature mismatch at expected[%d][%d]", i, j)
								}
							})
						}

						n, err := r.Data(nil)
						if !knownFeatureCount {
							assert.NoError(t, err)
						} else {
							assert.ErrorIs(t, err, io.EOF)
						}
						assert.Equal(t, 0, n)

						err = r.Close()
						assert.NoError(t, err)
					})
				})
			}

		})
	})
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
