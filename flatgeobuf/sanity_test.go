// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"io"
	"os"
	"testing"
	"unsafe"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	"github.com/gogama/flatgeobuf/packedrtree"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanity(t *testing.T) {
	t.Run("HilbertSort", func(t *testing.T) {
		// Sanity test that, somewhat indirectly, makes sure that our
		// implementation of packedrtree.HilbertSort agrees with the
		// canonical FlatGeobuf implementation as given by test data
		// files taken from the flatgeobuf project.
		for _, filename := range testDataFileNames(t) {
			t.Run(filename, func(t *testing.T) {
				f, err := os.Open(testDataRoot + filename)
				require.NoError(t, err)

				t.Cleanup(func() {
					err := f.Close()
					require.NoError(t, err)
				})

				// Skip unsupported versions.
				version, err := Magic(f)
				require.NoError(t, err)
				if version.Major < 3 {
					t.Log("Skipping file version", version.Major, version.Patch)
					return
				} else {
					_, err = f.Seek(0, io.SeekStart)
					require.NoError(t, err)
				}

				// Open FlatGeobuf file reader.
				r := NewFileReader(f)

				// Skip the header.
				_, err = r.Header()
				require.NoError(t, err)

				// Read the Index.
				index, err := r.Index()
				if err == ErrNoIndex {
					t.Log("Skipping file without index")
					return
				}
				require.NotNil(t, index)
				t.Log("I HAVE", index.NumRefs(), "REFS")

				// Serialize the index.
				var buf bytes.Buffer
				_, err = index.Marshal(&buf)
				require.NoError(t, err)

				// Get the raw index bytes.
				b := buf.Bytes()
				n, err := packedrtree.Size(index.NumRefs(), index.NodeSize())
				require.NoError(t, err)
				assert.Equal(t, n, len(b))

				// Get the sub-slice of index bytes that contains the leaf
				// nodes.
				size := int(unsafe.Sizeof(packedrtree.Ref{}))
				b = b[len(b)-index.NumRefs()*size:]

				// Read the byte slice into Refs.
				refs := make([]packedrtree.Ref, index.NumRefs())
				bounds := packedrtree.EmptyBox
				for i := range refs {
					refs[i].XMin = flatbuffers.GetFloat64(b[i*size+000:])
					refs[i].YMin = flatbuffers.GetFloat64(b[i*size+010:])
					refs[i].XMax = flatbuffers.GetFloat64(b[i*size+020:])
					refs[i].YMax = flatbuffers.GetFloat64(b[i*size+030:])
					refs[i].Offset = flatbuffers.GetInt64(b[i*size+040:])
					bounds.Expand(&refs[i].Box)
				}

				// Copy the Refs and Hilbert sort them.
				sorted := make([]packedrtree.Ref, len(refs))
				copy(sorted, refs)
				packedrtree.HilbertSort(sorted, bounds)

				// Verify the two slices are the same, thus ensuring
				// that our implementation and Hilbert sorting
				// produce the same results as the FlatGeobuf
				// implementation that wrote the file.
				//
				// NOTE: If this assertion starts failing, it could
				// mean there's a bug, or it could be related to the
				// fact that HilbertSort isn't a stable sort. If it
				// is the latter problem, I would be inclined to add
				// an exported packedrtree.HilbertSortStable
				// function mainly to enable this test to remain
				// viable and also because someone else might have a
				// use case for it.
				assert.Equal(t, refs, sorted)
			})
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		// Sanity test that ensures that round-tripping test data files
		// by reading their contents and writing the contents back to a
		// second file results in the second file having the same data
		// as the first file.
		testDataRunTests(t, func(t *testing.T, r1 *FileReader, filename string) {
			var b bytes.Buffer
			var err error

			var hdr1 *flat.Header
			var index1 *packedrtree.PackedRTree
			var data1 []flat.Feature

			t.Run("Read to Write", func(t *testing.T) {
				w := NewFileWriter(&b)
				var n int

				hdr1, err = r1.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr1)
				n, err = w.Header(hdr1)
				require.NoError(t, err)
				require.Greater(t, n, 0)

				index1, err = r1.Index()
				if err == ErrNoIndex {
					t.Logf("testdata file %q has no index on first read", filename)
					require.Nil(t, index1)
				} else {
					require.NoError(t, err)
					require.NotNil(t, index1)
					assert.Equal(t, hdr1.FeaturesCount(), uint64(index1.NumRefs()))
					assert.Equal(t, hdr1.IndexNodeSize(), index1.NodeSize())
					n, err = w.Index(index1)
					require.NoError(t, err)
				}

				data1, err = r1.DataRem()
				require.NoError(t, err)
				require.NotNil(t, data1)
				n, err = w.Data(data1)
				require.NoError(t, err)

				err = r1.Close()
				require.NoError(t, err)
				err = w.Close()
				require.NoError(t, err)
			})

			var hdr2 *flat.Header
			var index2 *packedrtree.PackedRTree
			var data2 []flat.Feature

			t.Run("Read Again", func(t *testing.T) {
				r2 := NewFileReader(&b)

				hdr2, err = r2.Header()
				require.NoError(t, err)
				require.NotNil(t, hdr2)
				mh1 := mockHeaderFromFlatBufferTable(hdr1)
				mh2 := mockHeaderFromFlatBufferTable(hdr2)
				assert.Equal(t, mh1, mh2)

				index2, err = r2.Index()
				if err == ErrNoIndex {
					t.Logf("testdata file %q has no index on second read", filename)
					require.Nil(t, index2)
				} else {
					require.NoError(t, err)
					require.NotNil(t, index2)
					assert.Equal(t, hdr2.FeaturesCount(), uint64(index2.NumRefs()))
					assert.Equal(t, hdr2.IndexNodeSize(), index2.NodeSize())
					assert.Equal(t, serializeIndex(t, index1), serializeIndex(t, index2))
				}

				data2, err = r2.DataRem()
				require.NoError(t, err)
				require.NotNil(t, data2)
				mf1 := mockFeaturesFromFlatBufferTable(data1)
				mf2 := mockFeaturesFromFlatBufferTable(data2)
				assert.Equal(t, mf1, mf2)
			})
		}, seekable|notSeekable)
	})
}

func serializeIndex(t *testing.T, index *packedrtree.PackedRTree) []byte {
	var b bytes.Buffer
	n, err := index.Marshal(&b)
	require.NoError(t, err)
	require.Greater(t, n, 0)
	return b.Bytes()
}
