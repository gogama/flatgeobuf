package packedrtree

// Result is a single index search result. A Result's fields can be used
// to locate the corresponding feature in the main data.
type Result struct {
	// Offset is the result feature's byte offset in the data section.
	Offset int
	// Index is the result feature's feature number.
	Index int
}
