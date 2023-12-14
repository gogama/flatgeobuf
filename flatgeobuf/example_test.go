// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf_test

import (
	"bytes"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	"github.com/gogama/flatgeobuf/packedrtree"
	flatbuffers "github.com/google/flatbuffers/go"
)

//go:embed testdata/flatgeobuf/*.fgb
var testFiles embed.FS

func openFile(name string) fs.File {
	f, err := testFiles.Open(name)
	if err != nil {
		panic(err)
	}
	return f
}

func ExampleMagic() {
	f := openFile("testdata/flatgeobuf/poly00.fgb")
	defer f.Close()

	version, err := flatgeobuf.Magic(f)
	fmt.Printf("%+v, %v\n", version, err)
	// Output: {Major:3 Patch:0}, <nil>
}

func ExampleFileReader_emptyFile() {
	// This simple example reads a trivial, empty, FlatGeobuf file. It
	// opens the file, reads the FlatGeobuf header, attempts to read the
	// index (but gets an error because the file has no index), and
	// reads the data section, which contains no features.

	r := flatgeobuf.NewFileReader(openFile("testdata/flatgeobuf/empty.fgb"))
	defer r.Close()

	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(flatgeobuf.HeaderString(hdr))

	index, err := r.Index()
	fmt.Printf("Index = %v, err = %v\n", index, err)

	data, err := r.DataRem()
	fmt.Printf("Data = %v, err = %v\n", data, err)
	// Output: Header{Name:gps_mobile_tiles,Type:Polygon,NumColumns:6,NumFeatures:UNKNOWN,NO INDEX,CRS:{Org:EPSG,Code:4326,Name:WGS 84,WKT:821 bytes}}
	// Index = <nil>, err = flatgeobuf: no index
	// Data = [], err = <nil>
}

func ExampleFileReader_unknownFeatureCount() {
	// This example reads a FlatGeobuf file which has an unknown feature
	// count, indicated by a zero in the header's feature count field.
	// The FileReader's DataRem() method provides a one-liner read all
	// available features at once. It is equivalent to using Data() in
	// a loop until EOF is reached.

	r := flatgeobuf.NewFileReader(openFile("testdata/flatgeobuf/unknown_feature_count.fgb"))
	defer r.Close()

	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(flatgeobuf.HeaderString(hdr))

	data, _ := r.DataRem() // Ignoring error to simplify example only!
	if len(data) > 0 {
		fmt.Printf("len(Data) -> %d, Data[0] -> %s\n", len(data), flatgeobuf.FeatureString(&data[0], hdr))
	}
	// Output: Header{Name:gps_mobile_tiles,Type:Polygon,NumColumns:6,NumFeatures:UNKNOWN,NO INDEX,CRS:{Org:EPSG,Code:4326,Name:WGS 84,WKT:821 bytes}}
	// len(Data) -> 1, Data[0] -> Feature{Geometry:{Type:Unknown,Bounds:[-69.911499,18.458768,-69.906006,18.463979]},Properties:{quadkey:0322113021201023,avg_d_kbps:16109,avg_u_kbps:11204,avg_lat_ms:36,tests:98,devices:49}}
}

func ExampleFileReader_Index() {
	// This example reads from a FlatGeobuf file which contains an
	// index. It reads the entire index data structure into memory using
	// the FileReader's Index() method, searches the index to find
	// candidate features that may intersect a bounding box, then
	// reads the data section up to the first candidate feature and
	// prints a string summary of the candidate.

	r := flatgeobuf.NewFileReader(openFile("testdata/flatgeobuf/countries.fgb"))
	defer r.Close()

	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(flatgeobuf.HeaderString(hdr))

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
		data := make([]flat.Feature, results[0].RefIndex+1)
		n, _ := r.Data(data) // Ignoring error to simplify example only!
		if n > results[0].RefIndex {
			fmt.Printf("First Result: %s\n", flatgeobuf.FeatureString(&data[results[0].RefIndex], hdr))
		}
	}
	// Output: Header{Name:countries,Envelope:[-180,-85.609038,180,83.64513],Type:MultiPolygon,NumColumns:2,NumFeatures:179,NodeSize:16,CRS:{Org:EPSG,Code:4326,Name:WGS 84,WKT:354 bytes}}
	// Index -> PackedRTree{Bounds:[-180,-85.609038,180,83.64513],NumRefs:179,NodeSize:16}
	// Results -> [{Offset:160424 RefIndex:165}]
	// First Result: Feature{Geometry:{Type:MultiPolygon,Bounds:[-171.79111,18.91619,-66.96466,71.357764]},Properties:{id:USA,name:United States of America}}
}

func ExampleFileReader_IndexSearch_streaming() {
	// This example demonstrates a streaming index search of a
	// FlatGeobuf file which contains an index. The FileReader's
	// IndexSearch() function reads and searches the index and fetches
	// all candidate features in a streaming manner, reading only the
	// minimum necessary data into memory.

	r := flatgeobuf.NewFileReader(openFile("testdata/flatgeobuf/UScounties.fgb"))
	defer r.Close()

	hdr, err := r.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(flatgeobuf.HeaderString(hdr))

	var data []flat.Feature

	// First search: Cook County, IL.
	if data, err = r.IndexSearch(packedrtree.Box{
		XMin: -87.63429124101445, YMin: 41.87174069508944,
		XMax: -87.61485750565028, YMax: 41.88406678494189,
	}); err != nil || len(data) == 0 {
		panic(fmt.Sprintf("err=  %v, len(data) = %d", err, len(data)))
	}
	fmt.Printf("First search, first Result: %s\n", flatgeobuf.FeatureString(&data[0], hdr))

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
	fmt.Printf("Second search, first Result: %s\n", flatgeobuf.FeatureString(&data[0], hdr))
	// Output: Header{Name:US__counties,Envelope:[-179.14734,17.884813,179.77847,71.352561],Type:Unknown,NumColumns:6,NumFeatures:3221,NodeSize:16,CRS:{Org:EPSG,Code:4269,Name:NAD83,WKT:1280 bytes}}
	// First search, first Result: Feature{Geometry:{Type:MultiPolygon,Bounds:[-88.263572,41.469555,-87.524044,42.154265]},Properties:{STATE_FIPS:17,COUNTY_FIP:031,FIPS:17031,STATE:IL,NAME:Cook,LSAD:County}}
	// Second search, first Result: Feature{Geometry:{Type:MultiPolygon,Bounds:[-113.33438,32.504938,-111.03991,34.04817]},Properties:{STATE_FIPS:04,COUNTY_FIP:013,FIPS:04013,STATE:AZ,NAME:Maricopa,LSAD:County}}
}

func ExamplePropReader() {
	// Start with a byte buffer containing three trivial properties.
	//
	// Normally you would obtain this buffer using the PropertiesBytes()
	// method of a flat.Feature, but we omit that part for simplicity.
	propBytes, _ := hex.DecodeString("000003000000666f6f010024082020020001")

	// Read the three properties. Error handling is omitted for brevity.
	//
	// If your column schema can vary, or you just want a simpler
	// interface to read properties, you may want to use the ReadSchema
	// method to read all properties at once.
	pr := flatgeobuf.NewPropReader(bytes.NewReader(propBytes))
	col, _ := pr.ReadUShort() // Column number
	str, _ := pr.ReadString() // Property value
	fmt.Printf("Column %d is the string value %q\n", col, str)
	col, _ = pr.ReadUShort() // Column number
	u, _ := pr.ReadUInt()    // Property value
	fmt.Printf("Column %d is the unsigned integer value 0x%x\n", col, u)
	col, _ = pr.ReadUShort() // Column number
	b, _ := pr.ReadBool()    // Property value
	fmt.Printf("Column %d is the boolean value %t\n", col, b)

	// Output: Column 0 is the string value "foo"
	// Column 1 is the unsigned integer value 0x20200824
	// Column 2 is the boolean value true
}

func simpleHeader() *flat.Header {
	// Create a file-level schema using a flat.Header. This is just to
	// facilitate the example ReadSchema code below. Normally you would
	// get the schema from the flat.Header of the file you are reading
	// or the current flat.Feature you are examining.
	bldr := flatbuffers.NewBuilder(0)

	col0Name := bldr.CreateString("A string")
	col1Name := bldr.CreateString("An unsigned int")
	col2Name := bldr.CreateString("A bool")

	flat.ColumnStart(bldr) // Column 0
	flat.ColumnAddName(bldr, col0Name)
	flat.ColumnAddType(bldr, flat.ColumnTypeString)
	col0 := flat.ColumnEnd(bldr)

	flat.ColumnStart(bldr) // Column 1
	flat.ColumnAddName(bldr, col1Name)
	flat.ColumnAddType(bldr, flat.ColumnTypeUInt)
	col1 := flat.ColumnEnd(bldr)

	flat.ColumnStart(bldr) // Column 2
	flat.ColumnAddName(bldr, col2Name)
	flat.ColumnAddType(bldr, flat.ColumnTypeBool)
	col2 := flat.ColumnEnd(bldr)

	flat.HeaderStartColumnsVector(bldr, 3)
	bldr.PrependUOffsetT(col2)
	bldr.PrependUOffsetT(col1)
	bldr.PrependUOffsetT(col0)
	cols := bldr.EndVector(3)

	flat.HeaderStart(bldr)
	flat.HeaderAddColumns(bldr, cols)
	hdr := flat.HeaderEnd(bldr)
	flat.FinishSizePrefixedHeaderBuffer(bldr, hdr)
	return flat.GetSizePrefixedRootAsHeader(bldr.FinishedBytes(), 0)
}

func ExamplePropReader_ReadSchema() {
	// Get an example FlatGeobuf file header. Both *flat.Header and
	// *flat.Feature implement Schema and can be used with ReadSchema.
	hdr := simpleHeader()

	// Start with a byte buffer containing three trivial properties
	// which follows the schema from the above header.
	//
	// Normally you would obtain this buffer using the PropertiesBytes()
	// method of a flat.Feature, but we omit that part for simplicity.
	propBytes, _ := hex.DecodeString("000003000000666f6f010024082020020001")

	// Read the properties.
	pr := flatgeobuf.NewPropReader(bytes.NewReader(propBytes))
	vals, _ := pr.ReadSchema(hdr)

	// Print the properties.
	fmt.Println(vals[0])
	fmt.Println(vals[1])
	fmt.Println(vals[2])
	// Output: PropValue{Name:"A string",Type:String,Value:"foo",ColIndex:0}
	// PropValue{Name:"An unsigned int",Type:UInt,Value:0x20200824,ColIndex:1}
	// PropValue{Name:"A bool",Type:Bool,Value:true,ColIndex:2}
}

func ExamplePropWriter() {
	var buf bytes.Buffer
	pw := flatgeobuf.NewPropWriter(&buf)

	// Serialize the properties to a byte buffer in the FlatGeobuf
	// properties format.
	pw.WriteUShort(0) // Column 0
	pw.WriteString("foo")
	pw.WriteUShort(1) // Column 1
	pw.WriteUInt(0x20200824)
	pw.WriteUShort(2) // Column 2
	pw.WriteBool(true)
	props := buf.Bytes()

	// Attach the properties to a FlatGeobuf feature. (Feature type and
	// geometry, which are required for a meaningful feature, are
	// omitted from this example to keep it lean.)
	bldr := flatbuffers.NewBuilder(0)
	propsOffset := bldr.CreateByteVector(props)
	flat.FeatureStart(bldr)
	flat.FeatureAddProperties(bldr, propsOffset)
	ftrOffset := flat.FeatureEnd(bldr)
	flat.FinishSizePrefixedFeatureBuffer(bldr, ftrOffset)

	fmt.Printf("props: %s, propsOffset: %d, ftrOffset: %d", hex.EncodeToString(props), propsOffset, ftrOffset)
	// Output: props: 000003000000666f6f010024082020020001, propsOffset: 24, ftrOffset: 32
}
