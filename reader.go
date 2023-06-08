package flatgeobuf

import (
	"github.com/gogama/flatgeobuf/packedrtree"
	"io"
)

type Reader struct {
	// TODO: I'd like this to be a lazy-ish reader where until you
	//       advance to the next section, the read pointer is at the
	//       end of the previous section.
}

func NewReader(r io.Reader) (*Reader, error) {
	// TODO: This will check magic number but not read ahead further.
}

func Header() (Flatgeobuf.Header, error) {
	// TODO :If not at header yet, advance to header, read, and cache it.
	// TODO: Return Header
}

func Index() (*packedrtree.PackedRTree, error) {
	// TODO: If not at index yet, advance to index, read, and cache it.
	// TODO: If no index, return nil and ErrNoIndex
	// TODO: If already passed index, return nil and ErrSkippedIndex
}

func Data() ([]Flatgeobuf.Feature, error) {
	// TODO: If not at data yet, advance to data and read it.
	// TODO: Otherwise the error is we're at the end.
	// TOdO: Implement it in terms of VisitData if possible.
}

func DataSearch(b packedrtree.Box) ([]Flatgeobuf.Feature, error) {

}

func DataVisit(v VisitFunc) error {
	// Instead of
}

func DataSearchVisit(b packedrtree.Box, v VisitFunc) error {

}

func (r *Reader) Close() error {
	// TODO: Close underlying.
}
