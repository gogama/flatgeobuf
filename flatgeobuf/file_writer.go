// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"io"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	"github.com/gogama/flatgeobuf/packedrtree"
)

// FileWriter writes a FlatGeobuf file to an underlying stream.
type FileWriter struct {
	stateful
	// w is the stream to write to.
	w io.Writer
	// numFeatures is the number of features recorded in the FlatGeobuf
	// header.
	numFeatures uint64
	// nodeSize is the index node size recorded in the FlatGeobuf
	// header.
	nodeSize uint16
	// featureIndex is the index of the next feature to write, a number
	// in the range [0, numFeatures]
	featureIndex uint64
}

// NewFileWriter creates a new FlatGeobuf file writer based on an
// underlying output stream.
//
// The underlying writer must be positioned at the beginning of the
// file, i.e. right before the FlatGeobuf magic number.
func NewFileWriter(w io.Writer) *FileWriter {
	if w == nil {
		textPanic("nil writer")
	}
	return &FileWriter{w: w}
}

// Header writes the FlatGeobuf file magic number, followed by the given
// FlatGeobuf header structure. The total number of bytes written,
// including magic number and header bytes, is returned.
//
// The input header table must be a size-prefixed root FlatBuffer table
// positioned at offset 0 within its FlatBuffer. This type of value is
// returned by FileReader.Header or from flat.GetSizePrefixedRootAsHeader.
//
// This method may only be called once, immediately after creating the
// FileWriter via NewFileWriter.
func (w *FileWriter) Header(hdr *flat.Header) (n int, err error) {
	// Minimally validate incoming pointer.
	if hdr == nil {
		textPanic("nil header")
	}

	// Cache feature count and check for overflow.
	var numFeatures uint64
	err = safeFlatBuffersInteraction(func() error {
		numFeatures = hdr.FeaturesCount()
		return nil
	})
	if err != nil {
		err = wrapErr("failed to get header feature count", err)
		return
	}

	// Cache index node size and check for illegal value.
	var nodeSize uint16
	err = safeFlatBuffersInteraction(func() error {
		nodeSize = hdr.IndexNodeSize()
		return nil
	})
	if err != nil {
		err = wrapErr("failed to get header index node size", err)
		return
	}
	if nodeSize == 1 {
		err = textErr("index node size may not be 1")
		return
	}

	// Transition into state for writing magic number.
	if err = w.toState(uninitialized, beforeMagic, outside); err == errUnexpectedState {
		err = textErr(errHeaderAlreadyCalled)
		return
	} else if err != nil {
		return
	}

	// Write the magic number.
	m, err := w.w.Write(magic[:])
	n += m
	if err != nil {
		err = w.toErr(wrapErr("failed to write magic number", err))
		return
	}

	// Transition into state for writing header.
	_ = w.toState(beforeMagic, beforeHeader, inside)

	// Write the header table.
	m, err = writeSizePrefixedTable(w.w, hdr.Table())
	n += m
	if err != nil {
		err = w.toErr(wrapErr("failed to write header", err))
		return
	}

	// Save cached feature count and index node size.
	w.numFeatures = numFeatures
	w.nodeSize = nodeSize

	// Transition into the state for writing index.
	_ = w.toState(beforeHeader, afterHeader, inside)

	// Successfully wrote header.
	return
}

// Index serializes and writes an in-memory FlatGeobuf index data
// structure to the index section of a FlatGeobuf file. The index node
// size and feature count must match the corresponding header fields
// written with Header. The total number of bytes written is returned.
//
// If used, this method must be called immediately after a successful
// call to Header, and may only be called once. Alternatively, the
// IndexData method may be used, or the index may be skipped and Data
// may be called directly after Header.
func (w *FileWriter) Index(index *packedrtree.PackedRTree) (n int, err error) {
	if index == nil {
		textPanic("nil index")
	}
	if err = w.canWriteIndex(); err != nil {
		return
	}
	return w.index(index)
}

// IndexData generates and writes an index for the given feature list,
// to the index section of a FlatGeobuf file, and then writes the
// features themselves into the data section. The input feature count
// must match the feature count header field written with Header. The
// total number of bytes written, to both index and data sections, is
// returned.
//
// If used, this method must be called immediately after a successful
// call to Header, and may only be called once. Alternatively, the Index
// method may be used if you already have an index data structure ready
// to serializeIndex, or the index may be skipped and Data may be called
// directly after Header.
//
// The input features are FlatBuffer tables. Each feature must be a
// size-prefixed root table positioned at offset 0 within its buffer.
// This type of value is returned by FileReader.Data,
// FileReader.DataRem, and from flat.GetSizePrefixedRootAsFeature.
func (w *FileWriter) IndexData(p []flat.Feature) (n int, err error) {
	// Verify state.
	if err = w.canWriteIndex(); err != nil {
		return
	}

	// Create index.
	refs := make([]packedrtree.Ref, len(p))
	bounds := packedrtree.EmptyBox
	var i int
	err = safeFlatBuffersInteraction(func() error {
		var offset int64
		for i = range p {
			refs[i].Offset = offset
			var size uint32
			if size, err = tableSize(p[i].Table()); err != nil {
				return wrapErr("feature %d", err, i)
			}
			err = featureBounds(&refs[i].Box, &p[i])
			if err != nil {
				return wrapErr("feature %d", err, i)
			}
			bounds.Expand(&refs[i].Box)
			offset += int64(size)
		}
		return nil
	})
	if err != nil {
		err = wrapErr("failed to index feature %d", err, i)
		return
	}
	packedrtree.HilbertSort(refs, bounds)
	var index *packedrtree.PackedRTree
	if index, err = packedrtree.New(refs, w.nodeSize); err != nil {
		return
	}

	// Write the index.
	if n, err = w.index(index); err != nil {
		return
	}

	// Write the data.
	var o int
	o, err = w.Data(p)
	n += o

	// Either all the features have been written, or an error occurred
	// writing a feature.
	return
}

// Data writes features into the data section. If the feature count
// field written with Header is non-zero, then the input feature count,
// plus the count of features already written, may not exceed file
// feature count. The total number of bytes written is returned.
//
// This method may only be called after Header has been called. If a
// positive index node size was indicated with Header, then it may only
// be called after Index has been called. This method may be called
// repeatedly to stream as many features as desired into the data
// section, as long as total number of features written does not exceed
// a positive feature count written with Header.
//
// The input features are FlatBuffer tables. Each feature must be a
// size-prefixed root table positioned at offset 0 within its buffer.
// This type of value is returned by FileReader.Data,
// FileReader.DataRem, and from flat.GetSizePrefixedRootAsFeature.
func (w *FileWriter) Data(p []flat.Feature) (n int, err error) {
	// Ensure we can fit all the requested features.
	if err = w.canWriteData(uint64(len(p))); err != nil {
		return
	}

	// Enter feature writing state.
	w.state = inData

	// Write each feature.
	for i := range p {
		var m int
		m, err = writeSizePrefixedTable(w.w, p[i].Table())
		n += m
		if err != nil {
			err = wrapErr("failed to write feature %d at data index %d", err, i, w.featureIndex)
			if m > 0 {
				_ = w.toErr(err)
			}
			return
		}
		w.featureIndex++
	}

	// Check for EOF.
	if w.featureIndex == w.numFeatures && w.numFeatures > 0 {
		_ = w.toState(inData, eof, inside)
	}

	// Return.
	return
}

// Close closes the FileWriter. All subsequent calls to any method will
// return ErrClosed.
//
// If the underlying stream implements io.Closer, this method invokes
// Close on the underlying stream and returns the result.
func (w *FileWriter) Close() error {
	if err := w.close(w.w); err != nil {
		return err
	} else if w.featureIndex < w.numFeatures {
		return fmtErr("truncated file: only wrote %d of %d header-indicated features", w.featureIndex, w.numFeatures)
	} else {
		return nil
	}
}

func (w *FileWriter) canWriteIndex() error {
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case uninitialized:
		return textErr(errHeaderNotCalled)
	case afterHeader:
		if w.nodeSize == 0 {
			return textErr(errHeaderNodeSizeZero)
		}
	case afterIndex, inData, eof:
		return textErr(errWritePastIndex)
	default:
		fmtPanic("logic error: unexpected state 0x%x looking to write index", w.state)
	}
	return nil
}

func (w *FileWriter) index(index *packedrtree.PackedRTree) (n int, err error) {
	// Transition into state for writing index.
	w.state = beforeIndex

	// Ensure index parameters agree with header parameters.
	if w.numFeatures != uint64(index.NumRefs()) {
		err = fmtErr("feature count mismatch (header=%d, index=%d)", w.numFeatures, index.NumRefs())
		w.state = afterHeader // Go back to header state.
		return
	} else if w.nodeSize != index.NodeSize() {
		err = fmtErr("node size mismatch (header=%d, index=%d)", w.nodeSize, index.NodeSize())
		w.state = afterHeader // Go back to header state.
		return
	}

	// Write the index.
	n, err = index.Marshal(w.w)
	if err != nil {
		err = w.toErr(wrapErr("failed to write index", err))
		return
	}

	// Transition into state for writing data.
	_ = w.toState(beforeIndex, afterIndex, inside)
	return
}

func (w *FileWriter) canWriteData(n uint64) error {
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case uninitialized:
		return textErr(errHeaderNotCalled)
	case afterHeader:
		if w.nodeSize > 0 {
			return textErr(errIndexNotWritten)
		}
	case afterIndex, inData:
		break
	case eof:
		return fmtErr("all %d features indicated in header already written", w.numFeatures)
	default:
		fmtPanic("logic error: unexpected state 0x%x looking to write data", w.state)
	}
	if w.numFeatures > 0 && w.numFeatures-w.featureIndex < n {
		excess := n - (w.numFeatures - w.featureIndex)
		return fmtErr("%d of %d features indicated in header already written, writing %d more would create an excess of %d", w.featureIndex, w.numFeatures, n, excess)
	}
	return nil
}

func featureBounds(b *packedrtree.Box, f *flat.Feature) error {
	*b = packedrtree.EmptyBox
	return safeFlatBuffersInteraction(func() error {
		var g flat.Geometry
		if f.Geometry(&g) != nil {
			n := g.XyLength()
			for i := 0; i+1 < n; i += 2 {
				b.ExpandXY(g.Xy(i+0), g.Xy(i+1))
			}
		}
		return nil
	})
}
