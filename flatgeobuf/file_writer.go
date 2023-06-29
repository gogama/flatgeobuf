// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"io"
	"math"

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

// TODO: Docs
func NewFileWriter(w io.Writer) *FileWriter {
	if w == nil {
		textPanic("nil writer")
	}
	return &FileWriter{w: w}
}

// TODO: Docs
// TODO: BECAUSE FlatBuffers has such a horrendous serialization
//
//		story, there's no general-purpose way to reliably know given
//		a table what it's size is, and it might not even be contiguous
//		in the buffer. THEREFORE, we require that the incoming table
//		be a size-prefixed ROOT table existing at offset 0 in the buffer,
//	 which is of course true of what our FileReader returns, but is not
//	 generally true.
func (w *FileWriter) Header(h *Header) (n int, err error) {
	// Minimally validate incoming pointer.
	if h == nil {
		textPanic("nil header")
	}

	// Cache feature count and check for overflow.
	var numFeatures uint64
	err = safeFlatBuffersInteraction(func() error {
		numFeatures = h.FeaturesCount()
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
		nodeSize = h.IndexNodeSize()
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
	m, err = writeSizePrefixedTable(w.w, h.Table())
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

// TODO: Docs
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

// TODO: Docs
func (w *FileWriter) IndexData(data []Feature) (n int, err error) {
	dataPtr := make([]*Feature, len(data))
	for i := range data {
		dataPtr[i] = &data[i]
	}
	return w.IndexDataPtr(dataPtr)
}

// TODO: Docs
func (w *FileWriter) IndexDataPtr(data []*Feature) (n int, err error) {
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
func (w *FileWriter) Data(f *Feature) (n int, err error) {
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

// TODO: Docs
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

func featureBounds(b *packedrtree.Box, f *Feature) error {
	*b = packedrtree.EmptyBox
	return safeFlatBuffersInteraction(func() error {
		var g Geometry
		if f.Geometry(&g) != nil {
			n := g.XyLength()
			for i := 0; i < n; i += 2 {
				b.ExpandXY(g.Xy(i+0), g.Xy(i+1))
			}
		}
		return nil
	})
}
