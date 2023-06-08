package flatgeobuf

import (
	"io"

	"github.com/gogama/flatgeobuf/packedrtree"
)

type Reader struct {
	// TODO: I'd like this to be a lazy-ish reader where until you
	//       advance to the next section, the read pointer is at the
	//       end of the previous section.
}

func NewReader(r io.Reader) (*Reader, error) {
	// TODO: This will check magic number but not read ahead further.
}

func (r *Reader) Header() (Header, error) {
	// TODO :If not at header yet, advance to header, read, and cache it.
	// TODO: Return Header
}

func (r *Reader) Index() (*packedrtree.PackedRTree, error) {
	// TODO: If not at index yet, advance to index, read, and cache it.
	// TODO: If no index, return nil and ErrNoIndex
	// TODO: If already passed index, return nil and ErrSkippedIndex
}

func (r *Reader) Data(p []Feature) (int, error) {
	// TODO: If not at data yet, advance to data and read it.
	// TODO: Otherwise the error is we're at the end.
	// TOdO: Implement it in terms of VisitData if possible.

	// TODO: Implement this similar to reader.Read so that it is easy
	//       to consume the data section as a stream in arbitrary
	//       increments.
}

func (r *Reader) DataAll() ([]Feature, error) {
	// TODO: Convenience version of Data that will read everything
	//       remaining.
}

func (r *Reader) DataSearch(b packedrtree.Box) ([]Feature, error) {

}

func (r *Reader) Close() error {
	// TODO: Close underlying.
}
