package flatgeobuf

import (
	"io"
	"math"
	"sort"

	"github.com/gogama/flatgeobuf/packedrtree"
	flatbuffers "github.com/google/flatbuffers/go"
)

// Reader reads an underlying stream as a FlatGeobuf file.
//
// TODO: Write docs.
type Reader struct {
	state readState // TODO: can probably be factored into a super-struct called stateful
	err   error     // TODO: can probably be factored into a super-struct named stateful
	// r is the stream to read from. It may also implement io.Seeker,
	// enabling a wider range of behaviours, but is not required to.
	r io.Reader
	// numFeatures is the number of features recorded in the
	// FlatGeobuf header.
	numFeatures int
	// nodeSize is the index node size recorded in the FlatGeobuf
	// header.
	nodeSize uint16
	// indexOffset is the byte offset of the spatial index within the
	// file being read by r. It will only have a non-zero value if r
	// also implements io.Seeker.
	indexOffset int64
	// dataOffset is the byte offset of the data section containing the
	// actual features. It will only have a non-zero value if r also
	// implements io.Seeker.
	dataOffset int64
	// cachedIndex is a cached reference to the loaded spatial index.
	// It will only have a non-zero value if the index was explicitly
	// unmarshalled via the Index() method, or implicitly unmarshalled
	// via the DataSearch() method.
	cachedIndex *packedrtree.PackedRTree
	// featureIndex is the index of the next feature to read, a number
	// in the [0, numFeatures].
	featureIndex int
	// featureOffset is the offset into the data section of the next
	// feature to read, a non-negative integer.
	featureOffset int64
}

type readState int

const (
	uninitialized readState = 0x00
	invalid                 = 0x01
	beforeMagic             = 0x11
	beforeHeader            = 0x21
	afterHeader             = 0x22
	beforeIndex             = 0x31
	afterIndex              = 0x32
	inData                  = 0x42
	eof                     = 0x52
)

// NewReader creates a new FlatGeobuf reader based on an underlying
// reader.
func NewReader(r io.Reader) *Reader {
	if r == nil {
		textPanic("nil reader")
	}
	return &Reader{r: r}
}

// TODO: Write docs.
func (r *Reader) Header() (*Header, error) {
	// Transition into state for reading magic number.
	if err := r.toState(uninitialized, beforeMagic); err != nil {
		return nil, textErr("Header has already been called")
	} else if err != nil {
		return nil, err
	}

	// Verify the magic number.
	v, err := Magic(r.r)
	if err != nil {
		return nil, r.toErr(wrapErr("failed to read magic number", err))
	}
	if v.Major < MinSpecMajorVersion || v.Major > MaxSpecMajorVersion {
		return nil, r.toErr(fmtErr("magic number has unsupported major version %d", v.Major))
	}

	// Transition into state for reading header.
	if err = r.toState(beforeMagic, beforeHeader); err != nil {
		return nil, err
	}

	// Read the header length, which is a little-endian 4-byte unsigned
	// integer.
	b := make([]byte, 4)
	if _, err = io.ReadFull(r.r, b); err != nil {
		return nil, r.toErr(wrapErr("header length read error", err))
	}
	headerLen := flatbuffers.GetUint32(b)
	if headerLen < 4 {
		return nil, r.toErr(fmtErr("header length %d not big enough for FlatBuffer uoffset_t", headerLen))
	} else if headerLen > headerMaxLen {
		return nil, r.toErr(fmtErr("header length %d exceeds limit of %d bytes", headerLen, headerMaxLen))
	}

	// Read the header bytes.
	tbl := make([]byte, 4+headerLen)
	copy(tbl, b)
	if _, err = io.ReadFull(r.r, tbl[4:]); err != nil {
		return nil, r.toErr(wrapErr("failed to read header table (len=%d)", err, headerLen))
	}

	// Convert to FlatBuffer-based Header structure and get number of
	// features and size of index tree nodes.
	var hdr *Header
	var numFeatures uint64
	var nodeSize uint16
	if err = safeFlatBuffersInteraction(func() {
		hdr = GetSizePrefixedRootAsHeader(tbl, 0)
		numFeatures = hdr.FeaturesCount()
		nodeSize = hdr.IndexNodeSize()
	}); err != nil {
		return nil, err
	}

	// Avoid overflow on feature count, because we interact with it
	// as a signed integer with platform-specific bit size. If there's
	// an error here, we still return the header in case caller still
	// wants to interact with it.
	if numFeatures > math.MaxInt {
		return hdr, r.toErr(fmtErr("header feature count %d exceeds limit of %d features", numFeatures, math.MaxInt))
	}

	// Check for an invalid index node size. If there's an error here,
	// we still return the header in case caller wants to interact with
	// it.
	if nodeSize == 1 {
		return hdr, r.toErr(textErr("header index node size 1 not allowed"))
	}

	// Store feature count and node size.
	r.numFeatures = int(numFeatures)
	r.nodeSize = nodeSize

	// Transition into state for reading index.
	if err = r.toState(beforeHeader, afterHeader); err != nil {
		return nil, err
	}

	// Return the header.
	return hdr, nil
}

// TODO: Write docs.
func (r *Reader) Index() (*packedrtree.PackedRTree, error) {
	// Transition into state for reading index.
	if err := r.toState(afterHeader, beforeIndex); err == errUnexpectedState {
		return nil, r.indexStateErr(r.state)
	} else if err != nil {
		return nil, err
	}

	// If the node size is zero, there is no index and the reader is
	// already pointing at the data section.
	if r.nodeSize == 0 {
		return nil, r.toState(beforeIndex, afterIndex)
	}

	// If we know our underlying reader is seekable, we may cache its
	// io.Seeker interface.
	var s io.Seeker

	// This Index() read might be after a Rewind() after a prior Index()
	// call read and cached the index. In this case, we can seek the
	// read cursor forward to the data section and return the cached
	// index.
	if r.cachedIndex != nil {
		s = r.r.(io.Seeker)
		if _, err := s.Seek(r.dataOffset, io.SeekStart); err != nil {
			return nil, r.toErr(wrapErr("failed to seek past cached index", err))
		}
		if err := r.toState(beforeIndex, afterIndex); err != nil {
			return nil, err
		}
		return r.cachedIndex, nil
	}

	// This Index() call might be after a Rewind() call. The Rewind()
	// call just ensures we have an io.Seeker and resets our reader
	// state to afterIndex, but, trying to be as lazy as possible, it
	// doesn't actually seek. Do that now.
	if r.indexOffset > 0 {
		s = r.r.(io.Seeker)
		if _, err := s.Seek(r.indexOffset, io.SeekStart); err != nil {
			return nil, r.toErr(wrapErr("failed to seek to index section", err))
		}
	}

	// Since we know that the reader's position is at the start of the
	// index section, we save this for future reference in case the user
	// does a Rewind().
	if err := r.saveIndexOffset(s); err != nil {
		return nil, err
	}

	// Read the actual index.
	prt, err := packedrtree.Unmarshal(r.r, r.numFeatures, r.nodeSize)
	if err != nil {
		return nil, r.toErr(wrapErr("failed to read index", err))
	}

	// Cache the index for use after future Rewind().
	r.cachedIndex = prt

	// Transition into state for reading feature data.
	if err = r.toState(beforeIndex, afterIndex); err != nil {
		return nil, err
	}

	// Return the index.
	return prt, nil
}

// TODO: Write docs.
func (r *Reader) IndexSearch(b packedrtree.Box) ([]Feature, error) {
	// Searches are only allowed if the reader is positioned immediately
	// after the header, either as a result of a Rewind(), or because of
	// a successful call to Header() immediately before.
	if err := r.toState(afterHeader, beforeIndex); err == errUnexpectedState {
		return nil, r.indexStateErr(r.state)
	} else if err != nil {
		return nil, err
	} else if r.nodeSize == 0 {
		r.state = afterIndex
		return nil, ErrNoIndex
	}

	// Search the index.
	var sr []packedrtree.Result
	var rs io.ReadSeeker
	if rs, _ = r.r.(io.ReadSeeker); rs != nil {
		if r.cachedIndex != nil {
			// If the index was cached by a prior call to Index(), reuse
			// it and seek past the index.
			sr = r.cachedIndex.Search(b)
			if _, err := rs.Seek(r.dataOffset, io.SeekCurrent); err != nil {
				return nil, r.toErr(wrapErr("failed to skip past index", err))
			}
		} else {
			// Save the index position in case we need to rewind.
			if err := r.saveIndexOffset(rs); err != nil {
				return nil, err
			}
			// Attempt an efficient streaming search without reading
			// the whole index into memory.
			var err error
			if sr, err = packedrtree.Seek(rs, r.numFeatures, r.nodeSize, b); err != nil {
				return nil, r.toErr(wrapErr("failed to seek-search index", err))
			}
		}
	} else if r.cachedIndex == nil {
		// Force caching the index.
		if _, err := r.Index(); err != nil {
			return nil, err
		}
		sr = r.cachedIndex.Search(b)
	} else {
		textPanic("logic error: index should not be cached")
	}

	// If the search results did not come from streaming search, sort
	// them so their offsets are in file order.
	if r.cachedIndex != nil {
		sort.Slice(sr, func(i, j int) bool {
			return sr[i].Offset < sr[j].Offset
		})
	}

	// The reader's read cursor is now past the index and at the
	// start of the data section.
	if err := r.toState(beforeIndex, afterIndex); err != nil {
		return nil, err
	}
	if err := r.saveDataOffset(rs); err != nil {
		return nil, err
	}
	if err := r.toState(afterIndex, inData); err != nil {
		return nil, err
	}

	// Create a helper function to skip over unnecessary features.
	type skipFunc func(n int64) error
	var skip skipFunc
	if rs != nil {
		skip = func(n int64) error {
			_, err := rs.Seek(n, io.SeekCurrent)
			return err
		}
	} else {
		buf := make([]byte, discardBufferSize)
		skip = func(n int64) error {
			return discard(r.r, buf, n)
		}
	}

	// Traverse the data section collecting all the features included
	// in the search results.
	fs := make([]Feature, len(sr))
	for i := range sr {
		if sr[i].Offset > r.featureOffset {
			if err := skip(sr[i].Offset - r.featureOffset); err != nil {
				return nil, r.toErr(wrapErr("failed to skip to feature %d (data offset %d) for search result %d", err, sr[i].RefIndex, sr[i].Offset, i))
			}
		}
		var err error
		r.featureIndex = sr[i].RefIndex
		r.featureOffset = sr[i].Offset
		if err = r.readFeature(&fs[i]); err != nil {
			return nil, err
		}
	}

	// Put the reader into EOF state so that it is not possible to make
	// weird residual calls to Data() or DataAll() from the position of
	// the last feature read.
	if err := r.toState(inData, eof); err != nil {
		return nil, err
	}

	// All search results are mapped to data features.
	return fs, nil
}

// TODO: Write docs.
func (r *Reader) Data(p []Feature) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.state == afterHeader {
		if err := r.skipIndex(); err != nil {
			return 0, err
		}
	}

	if r.state == afterIndex {
		if err := r.saveDataOffset(nil); err != nil {
			return 0, err
		}
		r.state = inData
	}

	if r.state == eof {
		return 0, io.EOF
	}

	if r.state == uninitialized {
		return 0, errHeaderNotCalled
	}

	r.sanityCheckState()

	rem := r.numFeatures - r.featureIndex
	n := len(p)
	if n > rem {
		n = rem
	}
	for i := 0; i < n; i++ {
		if err := r.readFeature(&p[i]); err != nil {
			return i, err
		}
	}
	if n == rem {
		if err := r.toState(inData, eof); err != nil {
			return n, err
		}
		return n, io.EOF
	}
	return n, nil
}

// TODO: Write docs.
func (r *Reader) DataAll() ([]Feature, error) {
	rem := r.numFeatures - r.featureIndex
	p := make([]Feature, rem)
	n, err := r.Data(p)
	p = p[0:n]
	if err != nil && err != io.EOF {
		return p, err
	}
	if n != rem {
		fmtPanic("expected %d features but read %d", rem, n)
	}
	return p, nil
}

// TODO: Write docs.
func (r *Reader) Rewind() error {
	if r.err != nil {
		return r.err
	} else if r.state == afterHeader {
		return nil // No-Op
	}

	r.sanityCheckState()
	if r.state < afterHeader {
		return errHeaderNotCalled
	} else if r.indexOffset == 0 {
		return textErr("can't rewind: reader is not an io.Seeker")
	}

	// Reset state to just after reading the header, but lazily do not
	// seek. Actual seeking will be done by either Index() or one of the
	// Data family of methods, as appropriate.
	r.state = afterHeader
	r.featureIndex = 0
	r.featureOffset = 0
	return nil
}

// TODO: Write docs.
func (r *Reader) Close() error { // TODO: Factor part of this into stateful.close()
	if r.err == nil {
		return r.err
	}

	r.err = ErrClosed
	if c, ok := r.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

func (r *Reader) sanityCheckState() {
	if r.state&invalid == invalid {
		fmtPanic("logic error: invalid state 0x%x", r.state)
	}
}

func (r *Reader) toState(expected, to readState) (err error) { // TODO: Factor into stateful
	// Always fail if the reader's already in the error state.
	if r.err != nil {
		return r.err
	}

	// Happy path to state transition is when reader is in the expected
	// state.
	if r.state == expected {
		r.state = to
		return nil
	}

	// Check for bad internal state.
	r.sanityCheckState()

	// Indicate that the state transition is invalid.
	return errUnexpectedState
}

func (r *Reader) toErr(err error) error {
	if r.err != nil {
		textPanic("logic error: already in error state")
	}

	r.err = err
	return err
}

const errReadPastIndex = "read position is past index"

func (r *Reader) indexStateErr(state readState) error {
	switch state {
	case uninitialized:
		return errHeaderNotCalled
	case afterIndex, inData, eof:
		if r.featureIndex > 0 {
			return textErr(errReadPastIndex + " (reader is an io.Seeker though, try Rewind)")
		} else {
			return textErr(errReadPastIndex)
		}
	default:
		fmtPanic("logic error: unexpected state 0x%x looking for index", state)
		return nil
	}
}

func (r *Reader) skipIndex() error {
	// Transition into state for working with index.
	if err := r.toState(afterHeader, beforeIndex); err != nil {
		return err
	}

	// Seek or read to the correct position.
	if r.nodeSize > 0 {
		if r.dataOffset > 0 { // If we already know the data offset, seek to it.
			s := r.r.(io.Seeker)
			if _, err := s.Seek(r.dataOffset, io.SeekStart); err != nil {
				return r.toErr(err)
			}
		} else if s, ok := r.r.(io.Seeker); ok { // If we can seek past the index, do so.
			if err := r.saveIndexOffset(s); err != nil {
				return err
			}
			indexSize, err := packedrtree.Size(r.numFeatures, r.nodeSize)
			if err != nil {
				return r.toErr(err)
			}
			r.dataOffset = r.indexOffset + indexSize
			if _, err = s.Seek(r.dataOffset, io.SeekStart); err != nil {
				return r.toErr(err)
			}
		} else { // Our only choice is to read past the index.
			indexSize, err := packedrtree.Size(r.numFeatures, r.nodeSize)
			if err != nil {
				return r.toErr(err)
			}
			bufSize := discardBufferSize
			if indexSize < int64(bufSize) {
				bufSize = int(indexSize)
			}
			if err = discard(r.r, make([]byte, bufSize), indexSize); err != nil {
				return r.toErr(err)
			}
		}
	}

	// We're now in the correct position.
	return r.toState(beforeIndex, afterIndex)
}

func (r *Reader) saveIndexOffset(s io.Seeker) error {
	return r.saveGenericOffset(s, &r.indexOffset, "index")
}

func (r *Reader) saveDataOffset(s io.Seeker) error {
	return r.saveGenericOffset(s, &r.dataOffset, "data")
}

func (r *Reader) saveGenericOffset(s io.Seeker, offsetPtr *int64, name string) error {
	if *offsetPtr == 0 {
		if s == nil {
			if s, _ = r.r.(io.Seeker); s == nil {
				return nil
			}
		}
		offset, err := s.Seek(0, io.SeekCurrent)
		if err != nil {
			return r.toErr(wrapErr("failed to query %s offset", err, name))
		}
		*offsetPtr = offset
	}
	return nil
}

func (r *Reader) readFeature(f *Feature) (err error) {
	// Read the feature length, which is a little-endian 32-bit integer.
	b := make([]byte, 4)
	if _, err = io.ReadFull(r.r, b); err != nil {
		return r.toErr(wrapErr("feature[%d] length read error (offset %d)", err, r.featureIndex, r.featureOffset))
	}
	featureLen := flatbuffers.GetUint32(b)
	if featureLen < 4 {
		return r.toErr(fmtErr("feature[%d] length %d not big enough for FlatBuffer uoffset_t (offset %d)", r.featureIndex, featureLen, r.featureOffset))
	}

	// Read the feature table bytes.
	tbl := make([]byte, 4+featureLen)
	copy(tbl, b)
	if _, err = io.ReadFull(r.r, tbl[4:]); err != nil {
		return r.toErr(wrapErr("failed to read feature[%d] (offset %d, len=%d)", err, r.featureIndex, r.featureOffset, featureLen))
	}

	// Read the uoffset_t that prefixes the tables bytes and which tells
	// us where the data starts.
	tblOffset := flatbuffers.GetUOffsetT(tbl[4:])

	// Convert the feature table into a size-prefixed FlatBuffer which
	// is a table of type Feature.
	f.Init(tbl, 4+tblOffset)

	// Advance the feature index and feature offset.
	r.featureIndex++
	r.featureOffset += 4 + int64(featureLen)

	// Successful read of a feature.
	return nil
}

// discardBufferSize is the suggested buffer size to use with the
// discard function.
const discardBufferSize = 8096

// discard reads and discards n bytes from a reader using the given
// temporary buffer as a scratch space to read into. At the end of this
// function, the contents of the buffer are undefined.
func discard(r io.Reader, buf []byte, n int64) error {
	for n > 0 {
		var a int
		var err error
		if int(n) < len(buf) {
			a, err = r.Read(buf[0:n])
		} else {
			a, err = r.Read(buf)
		}
		if err != nil {
			return err
		}
		n -= int64(a)
	}
	return nil
}
