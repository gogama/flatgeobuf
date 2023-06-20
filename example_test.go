// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf_test

import (
	"fmt"
	"github.com/gogama/flatgeobuf"
	"github.com/gogama/flatgeobuf/packedrtree"
	"os"
	"sort"
)

func ExampleReader_Empty_File() {
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
	fmt.Printf("Header -> { FeaturesCount = %d, IndexNodeSize = %d, Title = %q }\n", hdr.FeaturesCount(), hdr.IndexNodeSize(), hdr.Title())

	index, err := r.Index()
	fmt.Printf("Index = %v, err = %v\n", index, err)

	features, err := r.DataRem()
	fmt.Printf("Data = %v, err = %v\n", features, err)

	// Output: Header -> { FeaturesCount = 0, IndexNodeSize = 0, Title = "" }
	// Index = <nil>, err = <nil>
	// Data = [], err = <nil>
}

func ExampleReader_Materialized_Index() {
	f, err := os.Open("testdata/flatgeobuf/countries.fgb")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := flatgeobuf.NewReader(f)
	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Header -> { FeaturesCount = %d, IndexNodeSize = %d, Title = %q }\n", hdr.FeaturesCount(), hdr.IndexNodeSize(), hdr.Title())

	// Read the index into memory. This is a good option if repeated index
	// searches are planned.
	index, _ := r.Index()
	fmt.Printf("Index -> { Bounds = %s, NumRefs = %d, NodeSize = %d }\n", index.Bounds(), index.NumRefs(), index.NodeSize())

	// Search the index for features intersecting a bounding box.
	results := index.Search(packedrtree.Box{
		XMin: -157.84076832853575, YMin: 21.270348544130442, // TODO: Switch to Gogama, ON.
		XMax: -157.8224676330033, YMax: 21.281955907519844,
	})
	fmt.Printf("Results -> %+v\n", results)

	// Read the search results, and print the properties for the first
	// intersecting result.
	if len(results) > 0 {
		sort.Sort(results)
		data := make([]flatgeobuf.Feature, results[0].RefIndex+1)
		n, err := r.Data(data)
		fmt.Printf("Data -> { n = %d, err = %v }\n", n, err)
		if n > results[0].RefIndex {
			fmt.Printf("First Result's property bytes -> %q\n", data[results[0].RefIndex].PropertiesBytes())
		}
	}
	// Output: Header -> { FeaturesCount = 179, IndexNodeSize = 16, Title = "" }
	// Index -> { Bounds = [-180, -85.609038, 180, 83.64513], NumRefs = 179, NodeSize = 16 }
	// Results -> [{Offset:160424 RefIndex:165}]
	// Data -> { n = 166, err = <nil> }
	// First Result's property bytes -> "\x00\x00\x03\x00\x00\x00USA\x01\x00\x18\x00\x00\x00United States of America"
}

func ExampleReader_Streaming_Search() {
	f, err := os.Open("testdata/flatgeobuf/UScounties.fgb")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := flatgeobuf.NewReader(f)
	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Header -> { FeaturesCount = %d, IndexNodeSize = %d, Title = %q }\n", hdr.FeaturesCount(), hdr.IndexNodeSize(), hdr.Title())

	var features []flatgeobuf.Feature

	// First search: Cook County, IL.
	if features, err = r.IndexSearch(packedrtree.Box{
		XMin: -87.63429124101445, YMin: 41.87174069508944,
		XMax: -87.61485750565028, YMax: 41.88406678494189,
	}); err != nil || len(features) == 0 {
		panic(fmt.Sprintf("err=  %v, len(features) = %d", err, len(features)))
	}
	fmt.Printf("First search: %s (%q)\n", features[0].Geometry(&flatgeobuf.Geometry{}).Type(), features[0].PropertiesBytes())

	// Rewind.
	if err = r.Rewind(); err != nil {
		panic(err)
	}

	// Second search: Maricopa County, AZ.
	if features, err = r.IndexSearch(packedrtree.Box{
		XMin: -112.10457517582745, YMin: 33.43241637947986,
		XMax: -112.03936601127879, YMax: 33.46045877551812,
	}); err != nil || len(features) == 0 {
		panic(fmt.Sprintf("err=  %v, len(features) = %d", err, len(features)))
	}
	fmt.Printf("Second search: %s (%q)\n", features[0].Geometry(&flatgeobuf.Geometry{}).Type(), features[0].PropertiesBytes())

	// Output: Header -> { FeaturesCount = 3221, IndexNodeSize = 16, Title = "" }
	// First search: MultiPolygon ("\x00\x00\x02\x00\x00\x0017\x01\x00\x03\x00\x00\x00031\x02\x00\x05\x00\x00\x0017031\x03\x00\x02\x00\x00\x00IL\x04\x00\x04\x00\x00\x00Cook\x05\x00\x06\x00\x00\x00County")
	// Second search: MultiPolygon ("\x00\x00\x02\x00\x00\x0004\x01\x00\x03\x00\x00\x00013\x02\x00\x05\x00\x00\x0004013\x03\x00\x02\x00\x00\x00AZ\x04\x00\b\x00\x00\x00Maricopa\x05\x00\x06\x00\x00\x00County")
}

func ExampleReader_Unknown_Feature_Count() {
	// TODO
}
