package flatgeobuf

import (
	"io"

	"github.com/gogama/flatgeobuf/packedrtree"
)

type Writer struct {
}

func NewWriter(w io.Writer) (*Writer, error) {
	// TODO.
}

func Header(h *Header) error {

}

func Index(index *packedrtree.PackedRTree) error {
	// TODO: Header must be written.
	// TODO: Index node size must match same value in header.
	// TODO: Index feature count must match header.
}

func IndexData(data []Feature) error {
	// TODO: Header must be written and index may not be written.
	// TODO: Feature count must match header count.
	// TODO: Can't call this if index already written.
	// TODO: Can't call this if data already written.
}

func Data(data []Feature) error {
	// TODO: If index node size was set in header and no index written, BAD.
	// TODO: Cursory attempt to test that any Index written had same count.
	// TODO: Can't call this if data already written.
}
func (w *Writer) Close() error {
	// TODO
}
