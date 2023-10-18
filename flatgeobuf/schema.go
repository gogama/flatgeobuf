// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import "github.com/gogama/flatgeobuf/flatgeobuf/flat"

// Schema is a schema for FlatGeobuf's feature property format. It
// documents the number of available properties (ColumnsLength) for a
// feature and the metadata (flat.Column) for each property.
//
// Both the header structure (flat.Header) for a FlatGeobuf file and an
// individual feature within the data section (flat.Feature) implement
// Schema. When provided on the header table, the Schema applies to all
// features in the file except those that have their own schemas. When
// provided on an individual feature, the Schema applies only to that
// feature.
//
// Use PropReader.ReadSchema to read all properties from a FlatGeobuf
// properties buffer that is serialized according to a particular
// schema.
type Schema interface {
	// ColumnsLength returns the number of columns, i.e. properties, in
	// the schema.
	ColumnsLength() int
	// Columns obtains the metadata for the column, i.e. property, at a
	// specific index.
	//
	// The index j may range from 0 to ColumnsLength()-1. The pointer
	// obj must not be nil. On return, obj points to the property
	// metadata (flat.Column) for property j and the return value is
	// true. A return value of false indicates that no property metadata
	// was found for the property at index j.
	Columns(obj *flat.Column, j int) bool
}
