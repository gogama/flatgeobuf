// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package packedrtree provides the packed Hilbert R-Tree geospatial
// index data structure and search algorithms used in FlatGeobuf files.
//
// Although designed for FlatGeobuf, the simple, reusable, constructs
// within this package can be used standalone from FlatGeobuf, wherever
// a spatial index is needed.
package packedrtree
