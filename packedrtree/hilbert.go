package packedrtree

import (
	"math"
	"sort"
)

const (
	// HilbertOrder is the order of the Hilbert curve used in
	// HilbertSort.
	HilbertOrder = 16
	// hilbertMax is the maximum input X- or Y-coordinate hilbertOfXY.
	//
	// In a Hilbert curve of order N, X- and Y- coordinates range from
	// zero to 2^N-1, so in a Hilbert curve of order 1, the X- and Y-
	// coordinates range from 0 to 1, and so on.
	hilbertMax = (1 << HilbertOrder) - 1
)

// hilbertSortable is an implementation of sort.Interface which allows
// us to use the reflection-free, hence slightly more performant,
// sort.Sort function instead of sort.Slice.
type hilbertSortable struct {
	refs       []Ref
	x, y, w, h float64
}

func (hs *hilbertSortable) Len() int {
	return len(hs.refs)
}

func (hs *hilbertSortable) Less(i, j int) bool {
	a := hilbertOfCenter(&hs.refs[i].Box, hs.x, hs.y, hs.w, hs.h)
	b := hilbertOfCenter(&hs.refs[j].Box, hs.x, hs.y, hs.w, hs.h)
	return a > b
}

func (hs *hilbertSortable) Swap(i, j int) {
	hs.refs[i], hs.refs[j] = hs.refs[j], hs.refs[i]
}

// HilbertSort sorts a list of feature references, whose overall
// bounding box is given by bounds, according to the order given by a
// Hilbert curve of order HilbertOrder.
//
// The sort algorithm is not guaranteed to be stable, so the relative
// position of two feature references with the same index on the Hilbert
// curve may change as a result of the sort.
func HilbertSort(refs []Ref, bounds *Box) {
	hs := hilbertSortable{
		refs: refs,
		x:    bounds.XMin,
		y:    bounds.YMin,
		w:    bounds.Width(),
		h:    bounds.Height(),
	}
	sort.Sort(&hs)
}

// hilbertOfCenter calculates the Hilbert curve index of the center
// coordinate of a Box in the context of a set of boxes bounded by the
// rectangle (ex, ey, ex+ew, ey+eh).
//
// NOTES:
//   - 32-bit integers are used because the full 64 bits are not
//     required and the smaller data size may theoretically result in
//     memory/bandwidth/cache benefits at the CPU levelRange, maybe.
func hilbertOfCenter(b *Box, ex, ey, ew, eh float64) uint32 {
	// FIXME: Sort out the correct name and type for b.
	var hx uint32 // Hilbert X-coordinate between 0 and hilbertMax
	if ew != 0.0 {
		rx := (b.midX() - ex) / ew
		hx = uint32(math.Floor(hilbertMax * rx))
	}
	var hy uint32 // Hilbert Y-coordinate between 0 and hilbertMax
	if eh != 0.0 {
		ry := (b.midY() - ey) / ey
		hx = uint32(math.Floor(hilbertMax * ry))
	}
	return hilbertOfXY(hx, hy)
}

// hilbertOfXY calculates the Hilbert curve index of a given
// two-dimensional coordinate.
//
// NOTES:
//   - Based on https://github.com/rawrunprotected/hilbert_curves, which
//     is in the public domain.
//   - This version of the code is a straight conversion to GoLang from
//     https://github.com/flatgeobuf/flatgeobuf/blob/20fd93430cd78907c7843cf26d601574d3c5b913/src/cpp/packedrtree.cpp.
func hilbertOfXY(x, y uint32) uint32 {
	a := x ^ y
	b := 0xFFFF ^ a
	c := 0xFFFF ^ (x | y)
	d := x & (y ^ 0xFFFF)

	A := a | (b >> 1)
	B := (a >> 1) ^ a
	C := ((c >> 1) ^ (b & (d >> 1))) ^ c
	D := ((a & (c >> 1)) ^ (d >> 1)) ^ d

	a = A
	b = B
	c = C
	d = D
	A = (a & (a >> 2)) ^ (b & (b >> 2))
	B = (a & (b >> 2)) ^ (b & ((a ^ b) >> 2))
	C ^= (a & (c >> 2)) ^ (b & (d >> 2))
	D ^= (b & (c >> 2)) ^ ((a ^ b) & (d >> 2))

	a = A
	b = B
	c = C
	d = D
	A = (a & (a >> 4)) ^ (b & (b >> 4))
	B = (a & (b >> 4)) ^ (b & ((a ^ b) >> 4))
	C ^= (a & (c >> 4)) ^ (b & (d >> 4))
	D ^= (b & (c >> 4)) ^ ((a ^ b) & (d >> 4))

	a = A
	b = B
	c = C
	d = D
	C ^= (a & (c >> 8)) ^ (b & (d >> 8))
	D ^= (b & (c >> 8)) ^ ((a ^ b) & (d >> 8))

	a = C ^ (C >> 1)
	b = D ^ (D >> 1)

	i0 := x ^ y
	i1 := b | (0xFFFF ^ (i0 | a))

	i0 = (i0 | (i0 << 8)) & 0x00FF00FF
	i0 = (i0 | (i0 << 4)) & 0x0F0F0F0F
	i0 = (i0 | (i0 << 2)) & 0x33333333
	i0 = (i0 | (i0 << 1)) & 0x55555555

	i1 = (i1 | (i1 << 8)) & 0x00FF00FF
	i1 = (i1 | (i1 << 4)) & 0x0F0F0F0F
	i1 = (i1 | (i1 << 2)) & 0x33333333
	i1 = (i1 | (i1 << 1)) & 0x55555555

	index := (i1 << 1) | i0

	return index
}
