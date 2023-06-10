package flatgeobuf

import (
	"io"
	"math"
	"sort"

	"github.com/gogama/flatgeobuf/littleendian"
	"github.com/gogama/flatgeobuf/packedrtree"
)

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
}

type readState int

const (
	uninitialized readState = iota
	beforeMagic
	beforeHeader
	afterHeader
	beforeIndex
	afterIndex
	beforeData
	inData
)

func NewReader(r io.Reader) *Reader {
	if r == nil {
		textPanic("nil reader")
	}
	return &Reader{r: r}
}

func (r *Reader) Header() (*Header, error) {
	// Transition into state for reading magic number.
	if err := r.toState(uninitialized, beforeMagic); err != nil {
		return nil, err
	}

	// Verify the magic number.
	v, err := Magic(r.r)
	if err != nil {
		return nil, r.toError(wrapErr("failed to read magic number", err))
	}
	if v.Major < MinSpecMajorVersion || v.Major > MaxSpecMajorVersion {
		return nil, r.toError(fmtErr("magic number has unsupported major version %d", v.Major))
	}

	// Transition into state for reading header.
	if err = r.toState(beforeMagic, beforeHeader); err != nil {
		return nil, err
	}

	// Read the header length, which is a little-endian 4-byte unsigned
	// integer.
	b := make([]byte, 4)
	if _, err = io.ReadFull(r.r, b); err != nil {
		return nil, r.toError(wrapErr("failed to read header length", err))
	}
	headerLen := littleendian.Uint32(b)
	if headerLen > headerMaxLen {
		return nil, r.toError(fmtErr("header length %d exceeds limit of %d bytes", headerLen, headerMaxLen))
	}

	// Read the header bytes.
	b = make([]byte, headerLen)
	if _, err = io.ReadFull(r.r, b); err != nil {
		return nil, r.toError(wrapErr("failed to read %d header bytes", err, headerLen))
	}

	// Convert to FlatBuffer-based Header structure and get number of
	// features and size of index tree nodes.
	var hdr *Header
	var numFeatures uint64
	var nodeSize uint16
	if err = safeFlatBuffersInteraction(func() {
		hdr = GetRootAsHeader(b, 0)
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
		return hdr, r.toError(fmtErr("header feature count %d exceeds limit of %d features", numFeatures, math.MaxInt))
	}

	// Check for an invalid index node size. If there's an error here,
	// we still return the header in case caller wants to interact with
	// it.
	if nodeSize == 1 {
		return hdr, r.toError(textErr("header index node size 1 not allowed"))
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

func (r *Reader) Index() (*packedrtree.PackedRTree, error) {
	// Transition into state for reading index.
	if err := r.toState(afterHeader, beforeIndex); err != nil {
		return nil, err
	}

	// If the node size is zero, there is no index and the reader is
	// already pointing at the data section.
	if r.nodeSize == 0 {
		return nil, r.toState(beforeIndex, afterIndex)
	}

	// This Index() read might be after a Rewind() after a prior Index()
	// call read and cached the index. In this case, we can seek the
	// read cursor forward to the data section and return the cached
	// index.
	if r.cachedIndex != nil {
		rs := r.r.(io.Seeker)
		if _, err := rs.Seek(r.dataOffset, io.SeekStart); err != nil {
			return nil, r.toError(wrapErr("failed to seek to data section", err))
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
		rs := r.r.(io.Seeker)
		if _, err := rs.Seek(r.dataOffset, io.SeekStart); err != nil {
			return nil, r.toError(wrapErr("failed to seek to index section", err))
		}
	}

	// Since we know that the reader's position is at the start of the
	// index section, we save this for future reference in case the user
	// does a Rewind().
	if err := r.saveIndexOffset(); err != nil {
		return nil, err
	}

	// Read the actual index.
	prt, err := packedrtree.Unmarshal(r.r)
	if err != nil {
		return nil, r.toError(wrapErr("failed to read index", err))
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

func (r *Reader) IndexSearch(b packedrtree.Box) ([]Feature, error) {
	// Searches are only allowed if the reader is positioned immediately
	// after the header, either as a result of a Rewind(), or because of
	// a successful call to Header() immediately before.
	if r.state == afterHeader && r.nodeSize == 0 {
		return nil, ErrNoIndex
	} else if err := r.toState(afterHeader, beforeIndex); err != nil {
		return nil, err
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
				return nil, r.toError(wrapErr("failed to skip past index", err))
			}
		} else {
			// Save the index position in case we need to rewind.
			if err := r.saveIndexOffset(); err != nil {
				return nil, err
			}
			// Attempt an efficient streaming search without reading
			// the whole index into memory.
			var err error
			if err = r.saveIndexOffset(); err != nil {
				return nil, err
			} else if sr, err = packedrtree.Seek(rs, r.numFeatures, r.nodeSize, b); err != nil {
				return nil, r.toError(wrapErr("failed to seek-search index", err))
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
	if err := r.saveDataOffset(); err != nil {
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
		buf := make([]byte, 8096)
		skip = func(n int64) error {
			for n > 0 {
				var a int
				var err error
				if int(n) < len(buf) {
					a, err = r.r.Read(buf[0:n])
				} else {
					a, err = r.r.Read(buf)
				}
				if err != nil {
					return err
				}
				n -= int64(a)
			}
		}
	}

	// Traverse the data section collecting all the features included
	// in the search results.
	fs := make([]Feature, len(sr))
	var offset int64
	for i := range sr {
		if sr[i].Offset > offset {
			if err := skip(sr[i].Offset - offset); err != nil {
				return nil, r.toError(wrapErr("failed to skip to feature %d (data offset %d) for search result %d", err, sr[i].RefIndex, sr[i].Offset, i))
			}
		}
		var size int64
		var err error
		if size, err = r.readFeature(&fs[i]); err != nil {
			return nil, r.toError(wrapErr("failed to read feature %d (data offset %d) for search result %d", err, sr[i].RefIndex, sr[i].Offset))
		}
		r.featureIndex = sr[i].RefIndex
		offset = sr[i].Offset + size
	}

	// All search results are mapped to data features.
	return fs, nil

	// TODO: Skipping to tne end of the data section seems wrong for the
	//       slow non seekable case, but otherwise results in a weird
	//       thing where Data and DataAll can still be called. Should
	//       have a state for this.
}

func (r *Reader) Data(p []Feature) (int, error) {
	if err := r.checkDataState(); err != nil {
		return 0, err
	}
	rem := r.numFeatures - r.featureIndex
	n := len(p)
	if n > rem {
		n = rem
	}
	for i := 0; i < n; i++ {
		if _, err := r.readFeature(&p[i]); err != nil {
			return i, r.toError(wrapErr("failed to read feature %d", err, r.featureIndex))
		}
		r.featureIndex++
	}
	if n == rem {
		return n, io.EOF
	}
	return n, nil
}

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

func (r *Reader) Rewind() error {
	if r.err != nil {
		return r.err
	} else if r.state < afterHeader {
		// TODO: Return an appropriate error here.
	} else if r.state == afterHeader {
		return nil // No-Op
	} else if r.indexOffset == 0 {
		return textErr("can't rewind; reader is not an io.Seeker")
	}

	// Reset state to just after reading the header, but lazily do not
	// seek. Actual seeking will be done by either Index() or one of the
	// Data family of methods, as appropriate.
	r.state = afterHeader
	r.featureIndex = 0
	return nil
}

func (r *Reader) Close() error { // TODO: Factor part of this into stateful.close()
	if r.err == nil {
		return r.err
	}

	r.err = ErrClosed
	return nil
}

func (r *Reader) toState(expected, to readState) error { // TODO: Factor into stateful
	// Always fail if the reader's already in the error state.
	if r.err == nil {
		return r.err
	}

	// Happy path to state transition is when reader is in the expected
	// state.
	if r.state == expected {
		r.state = to
		return nil
	}

	// TODO: make nice contextual error messages based on the
	//       expected expected state.
	//  - As an example if desired state is beforeIndex but position
	//    is anything after afterHeader, then should return ErrPassedIndex.
	// - As another example, trying to do DataSearch if anywhere passed
	//   afterHeader could prompt an error that suggests Rewinding
}

func (r *Reader) toError(err error) error {
	if r.err != nil {
		textPanic("already in error state")
	}

	r.err = err
	return err
}

func (r *Reader) checkDataState() error {
	if r.err != nil {
		return r.err
	} else if r.state < afterHeader {
		// TODO: Return an appropriate error here.
	}

	if r.state == afterHeader {
		// TODO: Walk past index.
		r.state = afterIndex
	}

	if r.state == afterIndex {
		r.saveDataOffset()
		r.state = inData
	}

	if r.featureIndex == r.numFeatures {
		return io.EOF
	}

	return nil
}

func (r *Reader) saveIndexOffset() error {
	return r.saveGenericOffset(&r.indexOffset, "index")
}

func (r *Reader) saveDataOffset() error {
	return r.saveGenericOffset(&r.dataOffset, "data")
}

func (r *Reader) saveGenericOffset(offsetPtr *int64, name string) error {
	if *offsetPtr == 0 {
		if rs, ok := r.r.(io.Seeker); ok {
			offset, err := rs.Seek(0, io.SeekCurrent)
			if err != nil {
				return r.toError(wrapErr("failed to query %s offset", err, name))
			}
			*offsetPtr = offset
		}
	}
	return nil
}

func (r *Reader) readFeature(f *Feature) (size int64, err error) {
	// TODO: Read the size, uint32.

	// TODO: Read the feature.
}
