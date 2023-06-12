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

func (w *Writer) Header(h *Header) error {

}

func (w *Writer) Index(index *packedrtree.PackedRTree) error {
	// TODO: Header must be written.
	// TODO: Index node size must match same value in header.
	// TODO: Index feature count must match header.
}

func (w *Writer) IndexData(data []Feature) error {
	// TODO: Header must be written and index may not be written.
	// TODO: Feature count must be exactly equal to header count.
	// TODO: Can't call this if index already written as a PackedRTree.
	// TODO: Can't call this if data already written.
}

func (w *Writer) IndexDataPtr(data []*Feature) error {
	// TODO: Same as Data but with pointers, since that's more likely
	//       what people will find it easier to create with the existing
	//       FlatBuffers generated code.
}

func (w *Writer) Data(data []Feature) error {
	// TODO: If index node size was set in header and no index written, BAD.
	// TODO: Total feature count in all calls to this function must not
	//       exceed total count in header.
}

func (w *Writer) DataPtr(data []*Feature) error {
	// TODO: Same as Data but with pointers, since that's more likely
	//       what people will find it easier to create with the existing
	//       FlatBuffers generated code.
}

func (w *Writer) Close() error {
	// If already closed, error.
	// If header not written, error.
	// If index expected and not written, error.
	// If fewer features written than stated in header, error.
	// Otherwise, nil!
}
