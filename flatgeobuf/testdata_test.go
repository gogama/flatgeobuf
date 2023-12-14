// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const testDataRoot = "testdata/flatgeobuf/"

var testDataByteMap = make(map[string][]byte)

var testDataFileNamesSlice []string

var testDataFileNamesOnce sync.Once

type readerOnly struct {
	r *bytes.Reader
}

func (r *readerOnly) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func testDataFileNames(t *testing.T) []string {
	testDataFileNamesOnce.Do(func() {
		filesystem := os.DirFS(testDataRoot)
		err := fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".fgb") {
				testDataFileNamesSlice = append(testDataFileNamesSlice, path)
			}
			return nil
		})
		require.NoError(t, err, "failed to walk testdata directory")
	})
	return testDataFileNamesSlice
}

func newTestDataBytesReader(t *testing.T, seeker bool, filename string) io.Reader {
	b := testDataByteMap[filename]
	if b == nil {
		f, err := os.Open(testDataRoot + filename)
		require.NoError(t, err, "failed to open testdata file %q", filename)
		defer func() {
			_ = f.Close()
		}()
		b, err = io.ReadAll(f)
		require.NoError(t, err, "failed to fully read testdata file %q", filename)
		testDataByteMap[filename] = b
	}
	r := bytes.NewReader(b)
	if seeker {
		return r
	} else {
		return &readerOnly{r}
	}
}

func newTestDataFileReader(t *testing.T, seeker bool, filename string) *FileReader {
	return NewFileReader(newTestDataBytesReader(t, seeker, filename))
}

type runTestsFlag int

const (
	seekable           = 0x01
	notSeekable        = 0x02
	skipNoIndex        = 0x04
	includeUnsupported = 0x08
)

func testDataRunTests(t *testing.T, f func(t *testing.T, r *FileReader, filename string), flags runTestsFlag, filenames ...string) {
	wantSeekable := flags&seekable == seekable
	wantNotSeekable := flags&notSeekable == notSeekable
	if !wantSeekable && !wantNotSeekable {
		t.Error("at least one of seekable and notSeekable flags is required")
		t.FailNow()
	}
	if len(filenames) == 0 {
		filenames = testDataFileNames(t)
	}
	for i := range filenames {
		if flags&includeUnsupported != includeUnsupported {
			r := newTestDataBytesReader(t, false, filenames[i])
			version, err := Magic(r)
			require.NoError(t, err, "failed to read magic number for testdata file %q", filenames[i])
			if version.Major != 3 {
				t.Logf("Skipping testdata file %q with unsupported major version %d", filenames[i], version.Major)
				continue
			}
		}
		if flags&skipNoIndex == skipNoIndex {
			r := newTestDataFileReader(t, false, filenames[i])
			hdr, err := r.Header()
			require.NoError(t, err, "failed to read header for testdata file %q", filenames[i])
			if hdr.IndexNodeSize() == 0 {
				t.Logf("Skipping testdata file %q with because it has no index", filenames[i])
				continue
			}
		}
		t.Run(filenames[i], func(t *testing.T) {
			if wantSeekable {
				r := newTestDataFileReader(t, true, filenames[i])
				if wantNotSeekable {
					t.Run("Seekable", func(t *testing.T) {
						t.Cleanup(func() { _ = r.Close() })
						f(t, r, filenames[i])
					})
				} else {
					t.Cleanup(func() { _ = r.Close() })
					f(t, r, filenames[i])
				}
			}
			if wantNotSeekable {
				r := newTestDataFileReader(t, false, filenames[i])
				if wantSeekable {
					t.Run("Not Seekable", func(t *testing.T) {
						t.Cleanup(func() { _ = r.Close() })
						f(t, r, filenames[i])
					})
				} else {
					t.Cleanup(func() { _ = r.Close() })
					f(t, r, filenames[i])
				}
			}
		})
	}
}
