// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf_test

import (
	"fmt"
	"github.com/gogama/flatgeobuf"
	"os"
)

func ExampleReader_Empty() {
	f, err := os.Open("testdata/flatgeobuf/empty.fgb")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := flatgeobuf.NewReader(f)
	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Printf("features_count = %d, index_node_size = %d, title = %q\n", hdr.FeaturesCount(), hdr.IndexNodeSize(), hdr.Title())

	index, err := r.Index()
	fmt.Printf("index = %v, err = %v\n", index, err)

	features, err := r.DataAll()
	fmt.Printf("features = %v, err = %v\n", features, err)

	// Output: features_count = 0, index_node_size = 0, title = ""
	// index = <nil>, err = <nil>
	// features = [], err = <nil>
}
