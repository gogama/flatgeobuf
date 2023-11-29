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

func testDataBytesReader(t *testing.T, seeker bool, filename string) io.Reader {
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

func testDataFileReader(t *testing.T, seeker bool, filename string) *FileReader {
	return NewFileReader(testDataBytesReader(t, seeker, filename))
}

func testDataRunTests(t *testing.T, f func(t *testing.T, r *FileReader), seeker, skipUnsupported bool, filenames ...string) {
	if len(filenames) == 0 {
		filenames = testDataFileNames(t)
	}
	for i := range filenames {
		if skipUnsupported {
			r := testDataBytesReader(t, false, filenames[i])
			version, err := Magic(r)
			require.NoError(t, err, "failed to read magic number for testdata file %q", filenames[i])
			if version.Major != 3 {
				t.Logf("Skipping testdata file %q with unsupported major version %d", filenames[i], version.Major)
				continue
			}
		}
		r := testDataFileReader(t, seeker, filenames[i])
		t.Run(filenames[i], func(t *testing.T) {
			f(t, r)
		})
	}
}
