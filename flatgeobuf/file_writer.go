// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"io"
	"math"

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
	numFeatures int
	// nodeSize is the index node size recorded in the FlatGeobuf
	// header.
	nodeSize uint16
	// featureIndex is the index of the next feature to write, a number
	// in the range [0, numFeatures]
	featureIndex int
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
	if numFeatures > math.MaxInt {
		err = wrapErr("header feature count overflows int", err)
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
	if err = w.toState(uninitialized, beforeMagic); err == errUnexpectedState {
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
	if err = w.toState(beforeMagic, beforeHeader); err != nil {
		return
	}

	// Write the header table.
	m, err = writeSizePrefixedTable(w.w, hdr.Table())
	n += m
	if err != nil {
		err = w.toErr(wrapErr("failed to write header", err))
		return
	}

	// Save cached feature count and index node size.
	w.numFeatures = int(numFeatures)
	w.nodeSize = nodeSize

	// Transition into the state for writing index.
	err = w.toState(beforeHeader, afterHeader)

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
	if err = w.canWriteIndex(); err != nil {
		return
	}
	return w.index(index)
}

func (w *FileWriter) index(index *packedrtree.PackedRTree) (n int, err error) {
	// Transition into state for writing index.
	w.state = beforeIndex

	// Ensure index parameters agree with header parameters.
	if w.numFeatures != index.NumRefs() {
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
		err = w.toErr(err)
		return
	}

	// Transition into state for writing data.
	err = w.toState(beforeIndex, afterIndex)
	return
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
// to serialize, or the index may be skipped and Data may be called
// directly after Header.
//
// The input features are FlatBuffer tables. Each feature must be a
// size-prefixed root table positioned at offset 0 within its buffer.
// This type of value is returned by FileReader.Data,
// FileReader.DataRem, and from flat.GetSizePrefixedRootAsFeature.
func (w *FileWriter) IndexData(data []flat.Feature) (n int, err error) {
	dataPtr := make([]*flat.Feature, len(data))
	for i := range data {
		dataPtr[i] = &data[i]
	}
	return w.IndexDataPtr(dataPtr)
}

// TODO: Docs
// TODO: It's my preference to delete this and just support IndexData.
func (w *FileWriter) IndexDataPtr(data []*flat.Feature) (n int, err error) {
	// Verify state.
	if err = w.canWriteIndex(); err != nil {
		return
	}

	// Create index.
	refs := make([]packedrtree.Ref, len(data))
	bounds := packedrtree.EmptyBox
	var i int
	err = safeFlatBuffersInteraction(func() error {
		var offset int64
		for i = range data {
			refs[i].Offset = offset
			var size uint32
			if size, err = tableSize(data[i].Table()); err != nil {
				return wrapErr("feature %d", err, i)
			}
			err = featureBounds(&refs[i].Box, data[i])
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
	for i = range data {
		var o int
		o, err = w.Data(data[i])
		n += o
		if err != nil {
			return
		}
	}

	// Successfully wrote all the data.
	return
}

// TODO: Same issue as affecting Header and the IndexData* methods affects us
//
//	here: feature has to be a size-prefixed root table at offset 0
//
// FIXME: It would be simpler if this took a slice, and it would make a
//
//	better match with FileReader.
func (w *FileWriter) Data(f *flat.Feature) (n int, err error) {
	// Minimally validate incoming pointer.
	if f == nil {
		textPanic("nil feature")
	}

	// Ensure we can write another feature.
	if err = w.canWriteData(); err != nil {
		return
	}

	// Enter feature writing state.
	w.state = inData

	// Write the feature.
	if n, err = writeSizePrefixedTable(w.w, f.Table()); err != nil {
		err = wrapErr("failed to write feature %d", err, w.featureIndex)
		if n > 0 {
			_ = w.toErr(err)
		}
		return
	}
	w.featureIndex++

	// Check for EOF.
	if w.featureIndex == w.numFeatures && w.numFeatures > 0 {
		err = w.toState(inData, eof)
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
	case afterIndex, inData, eof /* TODO: Does EOF make sense? */ :
		return textErr(errWritePastIndex)
	default:
		fmtPanic("logic error: unexpected state 0x%x looking to write index", w.state)
	}
	return nil
}

func (w *FileWriter) canWriteData() error {
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
	return nil
}

func featureBounds(b *packedrtree.Box, f *flat.Feature) error {
	*b = packedrtree.EmptyBox
	return safeFlatBuffersInteraction(func() error {
		var g flat.Geometry
		if f.Geometry(&g) != nil {
			n := g.XyLength()
			for i := 0; i < n; i += 2 {
				b.ExpandXY(g.Xy(i+0), g.Xy(i+1))
			}
		}
		return nil
	})
}
