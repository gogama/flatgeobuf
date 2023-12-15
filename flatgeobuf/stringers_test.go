// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"testing"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHeaderString_Error(t *testing.T) {
	bldr := flatbuffers.NewBuilder(0)
	flat.HeaderStart(bldr)
	flat.HeaderAddFeaturesCount(bldr, 101)
	offset := flat.HeaderEnd(bldr)
	flat.FinishHeaderBuffer(bldr, offset)
	b := bldr.FinishedBytes()
	b[27] = 0xff // Corrupt the header's features_count field.
	hdr := flat.GetRootAsHeader(b, 0)

	actual := HeaderString(hdr)

	assert.Equal(t, "Header{error: panic: flatbuffers: runtime error: slice bounds out of range [65312:40]}", actual)
}

func TestHeaderString(t *testing.T) {
	testCases := []struct {
		name     string
		mh       mockHeader
		expected string
	}{
		{
			name:     "Zero",
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Name",
			mh: mockHeader{
				name: stringPtr("foo bar"),
			},
			expected: "Header{Name:foo bar,Type:Unknown,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Envelope",
			mh: mockHeader{
				envelope: []float64{0, 1, 2, 3},
			},
			expected: "Header{Envelope:[0,1,2,3],Type:Unknown,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Geometry Type",
			mh: mockHeader{
				geometryType: flat.GeometryTypeCompoundCurve,
			},
			expected: "Header{Type:CompoundCurve,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Flags.Z",
			mh: mockHeader{
				hasZ: true,
			},
			expected: "Header{Type:Unknown,Z,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Flags.Z",
			mh: mockHeader{
				hasZ: true,
			},
			expected: "Header{Type:Unknown,Z,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Flags.M",
			mh: mockHeader{
				hasM: true,
			},
			expected: "Header{Type:Unknown,M,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Flags.TM",
			mh: mockHeader{
				hasTM: true,
			},
			expected: "Header{Type:Unknown,TM,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Flags.All",
			mh: mockHeader{
				hasZ:  true,
				hasM:  true,
				hasT:  true,
				hasTM: true,
			},
			expected: "Header{Type:Unknown,Z|M|T|TM,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Columns.1",
			mh: mockHeader{
				columns: []mockColumn{
					{name: "foo"},
				},
			},
			expected: "Header{Type:Unknown,Columns:1,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Columns.2",
			mh: mockHeader{
				columns: []mockColumn{
					{name: "foo"},
					{name: "bar"},
				},
			},
			expected: "Header{Type:Unknown,Columns:2,Features:Unknown,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Features",
			mh: mockHeader{
				featuresCount: 1,
			},
			expected: "Header{Type:Unknown,Features:1,NodeSize:16,CRS:<nil>}",
		},
		{
			name: "Node Size",
			mh: mockHeader{
				indexNodeSize: uint16Ptr(4321),
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:4321,CRS:<nil>}",
		},
		{
			name: "CRS.Zero",
			mh: mockHeader{
				crs: &mockCRS{},
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:{Code:0,WKT:<nil>}}",
		},
		{
			name: "CRS.Org",
			mh: mockHeader{
				crs: &mockCRS{
					org: stringPtr("ham eggs"),
				},
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:{Org:ham eggs,Code:0,WKT:<nil>}}",
		},
		{
			name: "CRS.Code",
			mh: mockHeader{
				crs: &mockCRS{
					code: -1,
				},
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:{Code:-1,WKT:<nil>}}",
		},
		{
			name: "CRS.Name",
			mh: mockHeader{
				crs: &mockCRS{
					name: stringPtr("spam"),
				},
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:{Code:0,Name:spam,WKT:<nil>}}",
		},
		{
			name: "CRS.Description",
			mh: mockHeader{
				crs: &mockCRS{
					description: stringPtr("lorem ipsum"),
				},
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:{Code:0,Description:lorem ipsum,WKT:<nil>}}",
		},
		{
			name: "CRS.WKT",
			mh: mockHeader{
				crs: &mockCRS{
					wkt: stringPtr("dolor sit amet"),
				},
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:{Code:0,WKT:14 bytes}}",
		},
		{
			name: "CRS.Code String",
			mh: mockHeader{
				crs: &mockCRS{
					codeString: stringPtr("consectetur adipiscing elit"),
				},
			},
			expected: "Header{Type:Unknown,Features:Unknown,NodeSize:16,CRS:{Code:0,WKT:<nil>,CodeString:consectetur adipiscing elit}}",
		},
		{
			name: "All Fields",
			mh: mockHeader{
				name:         stringPtr("Collections"),
				envelope:     []float64{-2, -2, 1, 1},
				geometryType: flat.GeometryTypeGeometryCollection,
				hasZ:         true,
				hasM:         false,
				hasT:         true,
				hasTM:        false,
				columns: []mockColumn{
					{
						name: "example column",
					},
				},
				featuresCount: 1234,
				indexNodeSize: uint16Ptr(5),
				crs: &mockCRS{
					org:         stringPtr("EPSG"),
					code:        4326,
					name:        stringPtr("WGS 84"),
					description: stringPtr("World Geodetic System 1984 2D coordinate reference system"),
					wkt: stringPtr(`GEODCRS["WGS 84",
  DATUM["World Geodetic System 1984",
    ELLIPSOID["WGS 84", 6378137, 298.257223563, LENGTHUNIT["metre", 1]]],
  CS[ellipsoidal, 2],
    AXIS["Latitude (lat)", north, ORDER[1]],
    AXIS["Longitude (lon)", east, ORDER[2]],
    ANGLEUNIT["degree", 0.0174532925199433]]`),
				},
			},
			expected: "Header{Name:Collections,Envelope:[-2,-2,1,1],Type:GeometryCollection,Z|T,Columns:1,Features:1234,NodeSize:5,CRS:{Org:EPSG,Code:4326,Name:WGS 84,Description:World Geodetic System 1984 2D coordinate reference system,WKT:286 bytes}}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			hdr := testCase.mh.buildAsTable()

			actual := HeaderString(hdr)

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestFeatureString_Error(t *testing.T) {
	testCases := []struct {
		name        string
		schemaSetup func(ms *mockSchema)
	}{
		{
			name: "No Schema",
		},
		{
			name:        "Simple Schema",
			schemaSetup: func(ms *mockSchema) {},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			bldr := flatbuffers.NewBuilder(0)
			geometryOffset := bldr.StartVector(flatbuffers.SizeFloat64, 2, flatbuffers.SizeFloat64)
			bldr.PrependFloat64(-2)
			bldr.PrependFloat64(-3)
			bldr.EndVector(2)
			flat.FeatureStart(bldr)
			flat.FeatureAddGeometry(bldr, geometryOffset)
			offset := flat.FeatureEnd(bldr)
			flat.FinishFeatureBuffer(bldr, offset)
			b := bldr.FinishedBytes()
			b[11] = 0xff // Corrupt the feature's geometry field
			f := flat.GetRootAsFeature(b, 0)
			var ms *mockSchema
			var s Schema
			if testCase.schemaSetup != nil {
				ms = newMockSchema(t)
				testCase.schemaSetup(ms)
				s = ms
			}

			actual := FeatureString(f, s)

			assert.Equal(t, "Feature{error: geometry: panic: flatbuffers: runtime error: slice bounds out of range [16777220:32]}", actual)
			if ms != nil {
				ms.AssertExpectations(t)
			}
		})
	}
}

func TestFeatureString(t *testing.T) {
	testCases := []struct {
		name        string
		mf          mockFeature
		schemaSetup func(ms *mockSchema)
		expected    string
	}{
		{
			name:     "Zero",
			expected: "Feature{Geometry:<nil>,Properties:{}}",
		},
		{
			name: "Geometry.XY.Single",
			mf: mockFeature{
				geometry: &mockGeometry{
					xy: []float64{0},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:<nil>},Properties:{}}",
		},
		{
			name: "Geometry.XY.Odd",
			mf: mockFeature{
				geometry: &mockGeometry{
					xy: []float64{0, 1, 2},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:[0,1,0,1]},Properties:{}}",
		},
		{
			name: "Geometry.XY.Even",
			mf: mockFeature{
				geometry: &mockGeometry{
					xy: []float64{-10, 3, -10, 6, -5, 6, -5, 3, -10, 3},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:[-10,3,-5,6]},Properties:{}}",
		},
		{
			name: "Geometry.Ends",
			mf: mockFeature{
				geometry: &mockGeometry{
					ends: []uint32{3, 5},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:<nil>,Ends:2},Properties:{}}",
		},
		{
			name: "Geometry.Z",
			mf: mockFeature{
				geometry: &mockGeometry{
					z: []float64{-1, -2, 3},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:<nil>,Z:3},Properties:{}}",
		},
		{
			name: "Geometry.M",
			mf: mockFeature{
				geometry: &mockGeometry{
					m: []float64{11, 12},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:<nil>,M:2},Properties:{}}",
		},
		{
			name: "Geometry.T",
			mf: mockFeature{
				geometry: &mockGeometry{
					t: []float64{1},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:<nil>,T:1},Properties:{}}",
		},
		{
			name: "Geometry.TM",
			mf: mockFeature{
				geometry: &mockGeometry{
					tm: []uint64{1, 2, 3, 1000},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:<nil>,TM:4},Properties:{}}",
		},
		{
			name: "Geometry.Type",
			mf: mockFeature{
				geometry: &mockGeometry{
					geometryType: flat.GeometryTypeGeometryCollection,
				},
			},
			expected: "Feature{Geometry:{Type:GeometryCollection,Bounds:<nil>},Properties:{}}",
		},
		{
			name: "Geometry.Parts",
			mf: mockFeature{
				geometry: &mockGeometry{
					parts: []mockGeometry{
						{
							xy: []float64{-1, -1, 0, 0},
						},
						{
							xy: []float64{3, 3, 3, 4, 4, 4, 4, 3, 3, 3},
						},
					},
				},
			},
			expected: "Feature{Geometry:{Type:Unknown,Bounds:[-1,-1,4,4],Parts:2},Properties:{}}",
		},
		{
			name: "Feature Columns.No Property Bytes",
			mf: mockFeature{
				columns: []mockColumn{
					{
						name: "foo",
					},
				},
			},
			expected: "Feature{error: properties: flatgeobuf: failed to read column index (for property 0 of 1): EOF}",
		},
		{
			name: "Header Columns.No Property Bytes",
			mf:   mockFeature{},
			schemaSetup: func(ms *mockSchema) {
				ms.
					On("ColumnsLength").
					Return(1).
					Twice()
			},
			expected: "Feature{error: properties: flatgeobuf: failed to read column index (for property 0 of 1): EOF}",
		},
		{
			name: "Properties.No Columns",
			mf: mockFeature{
				properties: []byte{0x01},
			},
			expected: "Feature{Geometry:<nil>,Properties:{}}",
		},
		{
			name: "Properties.Empty Column Name",
			mf: mockFeature{
				properties: []byte{
					0x00, 0x00,
					0x11, 0x00, 0x00, 0x00,
					'e', 'm', 'p', 't', 'y', ' ', 'c', 'o', 'l', 'u', 'm', 'n', ' ', 'n', 'a', 'm', 'e',
				},
				columns: []mockColumn{
					{
						columnType: flat.ColumnTypeString,
					},
				},
			},
			expected: "Feature{Geometry:<nil>,Properties:{[0]:empty column name},Columns:1}",
		},
		{
			name: "Properties.Feature Schema",
			mf: mockFeature{
				properties: []byte{
					0x00, 0x00,
					0x27,
					0x01, 0x00,
					0x03, 0x00, 0x00, 0x00, 'b', 'a', 'z',
				},
				columns: []mockColumn{
					{
						name:       "foo",
						columnType: flat.ColumnTypeByte,
					},
					{
						name:       "bar",
						columnType: flat.ColumnTypeString,
					},
				},
			},
			expected: "Feature{Geometry:<nil>,Properties:{foo:39,bar:baz},Columns:2}",
		},
		{
			name: "Properties.Header Schema",
			mf: mockFeature{
				properties: []byte{
					0x00, 0x00,
					0x20, 0x20, 0x08, 0x24,
				},
			},
			schemaSetup: func(ms *mockSchema) {
				ms.
					On("ColumnsLength").
					Return(1).
					Twice()
				ms.
					On("Columns", mock.Anything, 0).
					Run(func(args mock.Arguments) {
						ptr := args[0].(*flat.Column)
						mc := mockColumn{
							name:       "hello",
							columnType: flat.ColumnTypeUInt,
						}
						*ptr = *mc.buildAsTable()
					}).
					Return(true).
					Once()
			},
			expected: "Feature{Geometry:<nil>,Properties:{hello:604512288}}",
		},
		{
			name: "Properties.Both Schemas",
			mf: mockFeature{
				properties: []byte{
					0x00, 0x00,
					0x20, 0x23, 0x04, 0x24,
				},
				columns: []mockColumn{
					{
						name:       "world",
						columnType: flat.ColumnTypeUInt,
					},
				},
			},
			schemaSetup: func(ms *mockSchema) { /* No methods are called. */ },
			expected:    "Feature{Geometry:<nil>,Properties:{world:604250912},Columns:1}",
		},
		{
			name: "Most Fields",
			mf: mockFeature{
				geometry: &mockGeometry{
					ends:         []uint32{3, 7},
					xy:           []float64{0, 0, 1, 1, 2, 2, 3, 3},
					z:            []float64{0, 1, 2, 3},
					t:            []float64{0, 1, 2, 3},
					tm:           []uint64{1000, 2000, 3000, 4000},
					geometryType: flat.GeometryTypeMultiLineString,
				},
				properties: []byte{
					0x01, 0x00,
					0x00,
					0x00, 0x00,
					0x01,
					0x02, 0x00,
					0xff, 0xff,
				},
			},
			schemaSetup: func(ms *mockSchema) {
				ms.
					On("ColumnsLength").
					Return(3).
					Twice()
				ms.
					On("Columns", mock.Anything, 0).
					Run(func(args mock.Arguments) {
						ptr := args[0].(*flat.Column)
						mc := mockColumn{
							name:       "flag1",
							columnType: flat.ColumnTypeBool,
						}
						*ptr = *mc.buildAsTable()
					}).
					Return(true).
					Once()
				ms.
					On("Columns", mock.Anything, 1).
					Run(func(args mock.Arguments) {
						ptr := args[0].(*flat.Column)
						mc := mockColumn{
							name:       "flag2",
							columnType: flat.ColumnTypeBool,
						}
						*ptr = *mc.buildAsTable()
					}).
					Return(true).
					Once()
				ms.
					On("Columns", mock.Anything, 2).
					Run(func(args mock.Arguments) {
						ptr := args[0].(*flat.Column)
						mc := mockColumn{
							name:       "short",
							columnType: flat.ColumnTypeShort,
						}
						*ptr = *mc.buildAsTable()
					}).
					Return(true).
					Once()
			},
			expected: "Feature{Geometry:{Type:MultiLineString,Bounds:[0,0,3,3],Ends:2,Z:4,T:4,TM:4},Properties:{flag2:false,flag1:true,short:-1}}",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			f := testCase.mf.buildAsTable()
			var ms *mockSchema
			var s Schema
			if testCase.schemaSetup != nil {
				ms = newMockSchema(t)
				testCase.schemaSetup(ms)
				s = ms
			}

			actual := FeatureString(f, s)

			assert.Equal(t, testCase.expected, actual)
			if ms != nil {
				ms.AssertExpectations(t)
			}
		})
	}
}
