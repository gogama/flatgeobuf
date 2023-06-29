// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree_test

import (
	"bytes"
	"fmt"

	"github.com/gogama/flatgeobuf/packedrtree"
)

// Create a Ref slice for example purposes.
var refs = []packedrtree.Ref{
	{Box: packedrtree.Box{XMin: -2, YMin: -2, XMax: -1, YMax: -1}, Offset: 0},
	{Box: packedrtree.Box{XMin: 1, YMin: 1, XMax: 2, YMax: 2}, Offset: 1},
	{Box: packedrtree.Box{XMin: -2, YMin: 1, XMax: -1, YMax: 2}, Offset: 2},
	{Box: packedrtree.Box{XMin: 1, YMin: -2, XMax: 2, YMax: -1}, Offset: 3},
}

func refsBounds(r []packedrtree.Ref) packedrtree.Box {
	b := packedrtree.EmptyBox // Important! Don't start with the zero box!
	for i := range r {
		b.Expand(&r[i].Box)
	}
	return b
}

func ExampleHilbertSort() {
	packedrtree.HilbertSort(refs, refsBounds(refs))

	fmt.Println(refs)
	// Output: [Ref{[1,-2,2,-1],Offset:3} Ref{[1,1,2,2],Offset:1} Ref{[-2,1,-1,2],Offset:2} Ref{[-2,-2,-1,-1],Offset:0}]
}

func ExampleNew() {
	packedrtree.HilbertSort(refs, refsBounds(refs)) // Refs must be Hilbert-sorted for New.
	index, _ := packedrtree.New(refs, 10)           // Ignore error ONLY to keep example simple.

	fmt.Println(index)
	// Output: PackedRTree{Bounds:[-2,-2,2,2],NumRefs:4,NodeSize:10}
}

func ExamplePackedRTree_Search() {
	packedrtree.HilbertSort(refs, refsBounds(refs)) // Refs must be Hilbert-sorted for New.
	index, _ := packedrtree.New(refs, 10)           // Ignore error ONLY to keep example simple.

	rs1 := index.Search(packedrtree.EmptyBox) // Search 1
	fmt.Println("Search 1:", rs1)

	rs2 := index.Search(packedrtree.Box{XMin: -10, YMin: -10, XMax: -5, YMax: -5}) // Search 2
	fmt.Println("Search 2:", rs2)

	rs3 := index.Search(index.Bounds()) // Search 3
	fmt.Printf("Search 3: %+v\n", rs3)

	rs4 := index.Search(packedrtree.Box{XMin: 0, YMin: -1, XMax: 1, YMax: 0}) // Search 4
	fmt.Printf("Search 4: %+v\n", rs4)
	// Output: Search 1: []
	// Search 2: []
	// Search 3: [{Offset:3 RefIndex:0} {Offset:1 RefIndex:1} {Offset:2 RefIndex:2} {Offset:0 RefIndex:3}]
	// Search 4: [{Offset:3 RefIndex:0}]
}

func ExampleUnmarshal() {
	// Marshal an index to bytes so that we can Unmarshal it.
	packedrtree.HilbertSort(refs, refsBounds(refs)) // Refs must be Hilbert-sorted for New.
	index, _ := packedrtree.New(refs, 10)           // Ignore error ONLY to keep example simple.
	var b bytes.Buffer
	_, _ = index.Marshal(&b)

	// Unmarshal from bytes.
	index, _ = packedrtree.Unmarshal(&b, len(refs), 10)
	fmt.Println(index)
	// Output: PackedRTree{Bounds:[-2,-2,2,2],NumRefs:4,NodeSize:10}
}

func ExampleSeek() {
	// Marshal an index to bytes so that we can seek within the raw bytes.
	packedrtree.HilbertSort(refs, refsBounds(refs)) // Refs must be Hilbert-sorted for New.
	index, _ := packedrtree.New(refs, 10)           // Ignore error ONLY to keep example simple.
	var b bytes.Buffer
	_, _ = index.Marshal(&b)

	// Do four streaming index searches on the raw index bytes.
	rs1, err1 := packedrtree.Seek(bytes.NewReader(b.Bytes()), len(refs), 10, packedrtree.EmptyBox)
	fmt.Println("Seek 1:", rs1, err1)

	rs2, err2 := packedrtree.Seek(bytes.NewReader(b.Bytes()), len(refs), 10, packedrtree.Box{XMin: -10, YMin: -10, XMax: -5, YMax: -5}) // Seek 2
	fmt.Println("Seek 2:", rs2, err2)

	rs3, err3 := packedrtree.Seek(bytes.NewReader(b.Bytes()), len(refs), 10, index.Bounds()) // Seek 3
	fmt.Printf("Seek 3: %+v %v\n", rs3, err3)

	rs4, err4 := packedrtree.Seek(bytes.NewReader(b.Bytes()), len(refs), 10, packedrtree.Box{XMin: 0, YMin: -1, XMax: 1, YMax: 0}) // Seek 4
	fmt.Printf("Seek 4: %+v %v\n", rs4, err4)
	// Output: Seek 1: [] <nil>
	// Seek 2: [] <nil>
	// Seek 3: [{Offset:3 RefIndex:0} {Offset:1 RefIndex:1} {Offset:2 RefIndex:2} {Offset:0 RefIndex:3}] <nil>
	// Seek 4: [{Offset:3 RefIndex:0}] <nil>
}
