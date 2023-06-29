// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/gogama/flatgeobuf/packedrtree"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Real tests.

func TestHilbertSort(t *testing.T) {
	// TODO: Real test cases.

	t.Run("Sanity", func(t *testing.T) {
		// Sanity test that, somewhat indirectly, makes sure that our
		// implementation of packedrtree.HilbertSort agrees with the
		// canonical FlatGeobuf implementation as given by test data
		// files taken from the flatgeobuf project.
		err := filepath.WalkDir("testdata/flatgeobuf/", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".fgb") {
				t.Run(path, func(t *testing.T) {
					f, err := os.Open(path)
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
					require.NoError(t, err)
					if index == nil {
						t.Log("Skipping file without index")
						return
					}
					t.Log("I HAVE", index.NumRefs(), "REFS")

					// Serialize the index.
					var buf bytes.Buffer
					_, err = index.Marshal(&buf)
					require.NoError(t, err)

					// Get the raw index bytes.
					b := buf.Bytes()
					n, err := packedrtree.Size(index.NumRefs(), index.NodeSize())
					require.NoError(t, err)
					assert.Equal(t, n, int64(len(b)))

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
			return nil
		})
		assert.NoError(t, err)
	})
}
