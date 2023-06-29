// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// hilbertInputs should be kept sorted in order of relative Hilbert
// number.
//
// ...	[B]                 ^                  [C]
// ...	                    |
// ...                      |
// ...                      |
// ...                      |
// ...                      |
// ...                      |
// ...                      |
// ... <--------------------+-------------------->
// ...                      | [D]
// ...                      |
// ...                      |
// ...                      |
// ...                      |
// ...                      |
// ...                      |                  [E]
// ... [A]                  v                  [F]
var hilbertInputs = []struct {
	name string
	b    Box
}{
	{"A", Box{-10, -10, -8, -8}},
	{"B", Box{-10, 8, -8, 10}},
	{"C", Box{8, 8, 10, 10}},
	{"D", Box{1, -2, 2, -1}},
	{"E", Box{8, -8, 10, -6}},
	{"F", Box{8, -10, 10, -8}},
}

var hilbertInputsBounds = EmptyBox

func init() {
	for i := range hilbertInputs {
		hilbertInputsBounds.Expand(&hilbertInputs[i].b)
	}
}

func TestHilbertSortable_Len(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		var zero hilbertSortable

		assert.Equal(t, 0, zero.Len())
	})

	t.Run("Value", func(t *testing.T) {
		value := hilbertSortable{refs: make([]Ref, 6), x: 0, y: 1, w: 2, h: 3}

		assert.Equal(t, 6, value.Len())
	})
}

func TestHilbertSortable_Less(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		hs := hilbertSortable{
			refs: make([]Ref, 1),
		}

		assert.False(t, hs.Less(0, 0))
	})

	t.Run("hilbertInputs", func(t *testing.T) {
		refs := make([]Ref, len(hilbertInputs))
		var bounds Box
		for i := range hilbertInputs {
			refs[i].Box = hilbertInputs[i].b
			refs[i].Offset = int64(i)
			bounds.Expand(&hilbertInputs[i].b)
		}
		hs := hilbertSortable{
			refs: refs,
			x:    bounds.XMin,
			y:    bounds.YMin,
			w:    bounds.Width(),
			h:    bounds.Height(),
		}

		for j := 0; j < len(hilbertInputs); j++ {
			for i := 0; i < j; i++ {
				t.Run(fmt.Sprintf("i=%d > j=%d", i, j), func(t *testing.T) {
					assert.True(t, hs.Less(j, i))
				})
			}

			t.Run(fmt.Sprintf("not(j<j), j=%d", j), func(t *testing.T) {
				assert.False(t, hs.Less(j, j))
			})

			for k := j + 1; k < len(hilbertInputs); k++ {
				t.Run(fmt.Sprintf("j=%d > k=%d", j, k), func(t *testing.T) {
					assert.True(t, hs.Less(k, j))
				})
			}
		}
	})
}

func TestHilbertSortable_Swap(t *testing.T) {
	t.Run("One", func(t *testing.T) {
		one := hilbertSortable{refs: make([]Ref, 1)}

		one.Swap(0, 0)

		assert.Equal(t, Ref{}, one.refs[0])
	})
	t.Run("Two", func(t *testing.T) {
		zero := Ref{}
		one := Ref{Box{1, 1, 1, 1}, 1}
		two := func() hilbertSortable {
			return hilbertSortable{
				refs: []Ref{
					zero,
					one,
				},
				x: 2,
				y: 2,
				w: 2,
				h: 2,
			}
		}

		t.Run("There", func(t *testing.T) {
			x := two()
			x.Swap(0, 1)

			assert.Equal(t, one, x.refs[0])
			assert.Equal(t, zero, x.refs[1])
		})

		t.Run("ThereAndBackAgain", func(t *testing.T) {
			x := two()
			x.Swap(1, 0)
			x.Swap(1, 0)

			assert.Equal(t, zero, x.refs[0])
			assert.Equal(t, one, x.refs[1])
		})
	})
}

func TestHilbertSort(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		var refs []Ref
		var bounds Box

		HilbertSort(refs, bounds)
	})

	t.Run("Singleton", func(t *testing.T) {
		ref := Ref{
			Box:    Box{XMin: -1, YMin: -1, XMax: 1, YMax: 1},
			Offset: 555,
		}
		refs := []Ref{ref}
		bounds := ref.Box

		HilbertSort(refs, bounds)

		assert.Equal(t, []Ref{ref}, refs)
	})

	t.Run("hilbertInputs", func(t *testing.T) {
		var refs []Ref
		var bounds Box
		for i := range hilbertInputs {
			refs = append(refs, Ref{
				Box:    hilbertInputs[i].b,
				Offset: int64(i),
			})
			bounds.Expand(&hilbertInputs[i].b)
		}

		HilbertSort(refs, bounds)

		isReverseSorted := sort.SliceIsSorted(refs, func(i, j int) bool {
			return refs[i].Offset > refs[i].Offset
		})
		assert.True(t, isReverseSorted, "Slice should be sorted by descending Hilbert index, but is not.")
	})
}

func TestHilbertOfCenter(t *testing.T) {
	t.Run("ZeroWidth", func(t *testing.T) {
		actual := hilbertOfCenter(&Box{0, 0, 0, 0}, 0, 0, 0, 10)

		assert.Equal(t, uint32(0), actual)
	})
	t.Run("ZeroHeight", func(t *testing.T) {
		actual := hilbertOfCenter(&Box{0, 0, 0, 0}, 0, 0, 10, 0)

		assert.Equal(t, uint32(0), actual)
	})
	t.Run("HilbertInputs", func(t *testing.T) {
		var i int
		var hi uint32
		for j := range hilbertInputs {
			hj := hilbertOfCenter(&hilbertInputs[j].b, hilbertInputsBounds.XMin, hilbertInputsBounds.YMin, hilbertInputsBounds.Width(), hilbertInputsBounds.Height())
			assert.Greater(t, hj, hi, "hilbertOfCenter(hilbertInputs[%d]) must be greater than hilbertOfCenter(hilbertInputs[%d])", j, i)
			i = j
			hi = hj
		}
	})
}

func TestHilbertOfXY(t *testing.T) {
	testCases := []struct {
		name     string
		x, y     uint32
		expected uint32
	}{
		{name: "Zero"},
		{name: "OneX", x: 1, y: 0, expected: 1},
		{name: "OneXY", x: 1, y: 1, expected: 2},
		{name: "OneY", x: 0, y: 1, expected: 3},
		{name: "HalfMaxX", x: math.MaxUint32 / math.MaxUint16, y: 0, expected: 0x30001},
		{name: "HalfMaxY", x: 0, y: math.MaxUint32 / math.MaxUint16, expected: 0x30003},
		{name: "HalfMaxXY", x: math.MaxUint32 / math.MaxUint16, y: math.MaxUint32 / math.MaxUint16, expected: 0xaaaaaaaa},
		{name: "MaxY", x: 0, y: math.MaxUint32, expected: 0xffff7777},
		{name: "MaxX", x: math.MaxUint32, y: 0, expected: 0xffffdddd},
		{name: "MaxXY", x: math.MaxUint32, y: math.MaxUint32, expected: 0xaaaaaaaa},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := hilbertOfXY(testCase.x, testCase.y)

			assert.Equal(t, testCase.expected, actual)
		})
	}
}
