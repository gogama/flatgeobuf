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

	r := flatgeobuf.NewFileReader(f)
	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(hdr)

	index, err := r.Index()
	fmt.Printf("Index = %v, err = %v\n", index, err)

	data, err := r.DataRem()
	fmt.Printf("Data = %v, err = %v\n", data, err)

	// Output: Header{Name:gps_mobile_tiles,Type:Polygon,NumColumns:6,NumFeatures:UNKNOWN,NO INDEX,CRS:{Org:EPSG,Code:4326,Name:WGS 84,WKT:821 bytes}}
	// Index = <nil>, err = <nil>
	// Data = [], err = <nil>
}

func ExampleReader_Materialized_Index() {
	f, err := os.Open("testdata/flatgeobuf/countries.fgb")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := flatgeobuf.NewFileReader(f)
	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(hdr)

	// Read the index into memory. This is a good option if repeated index
	// searches are planned.
	index, _ := r.Index()
	fmt.Println("Index ->", index)

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
		n, _ := r.Data(data) // Ignoring error to simplify example only!
		if n > results[0].RefIndex {
			fmt.Printf("First Result: %s\n", data[results[0].RefIndex].StringSchema(hdr))
		}
	}

	// Output: Header{Name:countries,Envelope:[-180,-85.609038,180,83.64513],Type:MultiPolygon,NumColumns:2,NumFeatures:179,NodeSize:16,CRS:{Org:EPSG,Code:4326,Name:WGS 84,WKT:354 bytes}}
	// Index -> PackedRTree{Bounds:[-180,-85.609038,180,83.64513],NumRefs:179,NodeSize:16}
	// Results -> [{Offset:160424 RefIndex:165}]
	// First Result: Feature{Geometry:{Type:MultiPolygon,Bounds:[-171.79111,18.91619,-66.96466,71.357764]},Properties:{id:USA,name:United States of America}}
}

func ExampleReader_Streaming_Search() {
	f, err := os.Open("testdata/flatgeobuf/UScounties.fgb")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := flatgeobuf.NewFileReader(f)
	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(hdr)

	var data []flatgeobuf.Feature

	// First search: Cook County, IL.
	if data, err = r.IndexSearch(packedrtree.Box{
		XMin: -87.63429124101445, YMin: 41.87174069508944,
		XMax: -87.61485750565028, YMax: 41.88406678494189,
	}); err != nil || len(data) == 0 {
		panic(fmt.Sprintf("err=  %v, len(data) = %d", err, len(data)))
	}
	fmt.Printf("First search, first Result: %s\n", data[0].StringSchema(hdr))

	// Rewind.
	if err = r.Rewind(); err != nil {
		panic(err)
	}

	// Second search: Maricopa County, AZ.
	if data, err = r.IndexSearch(packedrtree.Box{
		XMin: -112.10457517582745, YMin: 33.43241637947986,
		XMax: -112.03936601127879, YMax: 33.46045877551812,
	}); err != nil || len(data) == 0 {
		panic(fmt.Sprintf("err=  %v, len(features) = %d", err, len(data)))
	}
	fmt.Printf("Second search, first Result: %s\n", data[0].StringSchema(hdr))

	// Output: Header{Name:US__counties,Envelope:[-179.14734,17.884813,179.77847,71.352561],Type:Unknown,NumColumns:6,NumFeatures:3221,NodeSize:16,CRS:{Org:EPSG,Code:4269,Name:NAD83,WKT:1280 bytes}}
	// First search, first Result: Feature{Geometry:{Type:MultiPolygon,Bounds:[-88.263572,41.469555,-87.524044,42.154265]},Properties:{STATE_FIPS:17,COUNTY_FIP:031,FIPS:17031,STATE:IL,NAME:Cook,LSAD:County}}
	// Second search, first Result: Feature{Geometry:{Type:MultiPolygon,Bounds:[-113.33438,32.504938,-111.03991,34.04817]},Properties:{STATE_FIPS:04,COUNTY_FIP:013,FIPS:04013,STATE:AZ,NAME:Maricopa,LSAD:County}}
}

func ExampleReader_Unknown_Feature_Count() {
	f, err := os.Open("testdata/flatgeobuf/unknown_feature_count.fgb")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := flatgeobuf.NewFileReader(f)
	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(hdr)

	data, _ := r.DataRem() // Ignoring error to simplify example only!
	if len(data) > 0 {
		fmt.Printf("len(Data) -> %d, Data[0] -> %s\n", len(data), data[0].StringSchema(hdr))
	}

	// Output: Header{Name:gps_mobile_tiles,Type:Polygon,NumColumns:6,NumFeatures:UNKNOWN,NO INDEX,CRS:{Org:EPSG,Code:4326,Name:WGS 84,WKT:821 bytes}}
	// len(Data) -> 1, Data[0] -> Feature{Geometry:{Type:Unknown,Bounds:[-69.911499,18.458768,-69.906006,18.463979]},Properties:{quadkey:0322113021201023,avg_d_kbps:237,avg_u_kbps:196,avg_lat_ms:36,tests:98,devices:49}}
}
