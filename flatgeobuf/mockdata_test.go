// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"io"
	"math"
	"sync"
	"testing"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	"github.com/gogama/flatgeobuf/packedrtree"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/require"
)

type mockColumn struct {
	name        string
	columnType  flat.ColumnType
	title       *string
	description *string
	width       int32
	precision   int32
	scale       int32
	nullable    bool
	unique      bool
	primaryKey  bool
	metadata    *string
}

func mockColumnFromFlatBufferTable(col *flat.Column) (mc mockColumn) {
	mc.name = string(col.Name())
	mc.columnType = col.Type()
	if col.Title() != nil {
		mc.title = stringPtr(string(col.Title()))
	}
	if col.Description() != nil {
		mc.description = stringPtr(string(col.Description()))
	}
	mc.width = col.Width()
	mc.precision = col.Precision()
	mc.scale = col.Scale()
	mc.nullable = col.Nullable()
	mc.unique = col.Unique()
	mc.primaryKey = col.PrimaryKey()
	if col.Metadata() != nil {
		mc.metadata = stringPtr(string(col.Metadata()))
	}
	return
}

func (mc *mockColumn) build(bldr *flatbuffers.Builder) flatbuffers.UOffsetT {
	func() {
		offset := bldr.CreateString(mc.name)
		defer flat.ColumnAddName(bldr, offset)
		defer flat.ColumnAddType(bldr, mc.columnType)
		if mc.title != nil {
			offset = bldr.CreateString(*mc.title)
			defer flat.ColumnAddTitle(bldr, offset)
		}
		if mc.description != nil {
			offset = bldr.CreateString(*mc.description)
			defer flat.ColumnAddDescription(bldr, offset)
		}
		if mc.width != 0 {
			defer flat.ColumnAddWidth(bldr, mc.width)
		}
		if mc.precision != 0 {
			defer flat.ColumnAddPrecision(bldr, mc.precision)
		}
		if mc.scale != 0 {
			defer flat.ColumnAddScale(bldr, mc.scale)
		}
		if mc.nullable {
			defer flat.ColumnAddNullable(bldr, true)
		}
		if mc.unique {
			defer flat.ColumnAddUnique(bldr, true)
		}
		if mc.primaryKey {
			defer flat.ColumnAddPrimaryKey(bldr, true)
		}
		if mc.metadata != nil {
			offset = bldr.CreateString(*mc.metadata)
			defer flat.ColumnAddMetadata(bldr, offset)
		}
		flat.ColumnStart(bldr)
	}()
	return flat.ColumnEnd(bldr)
}

func buildMockColumns(bldr *flatbuffers.Builder, cols []mockColumn, startVector func(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	offsets := make([]flatbuffers.UOffsetT, len(cols))
	for i := range offsets {
		offsets[i] = cols[i].build(bldr)
	}
	startVector(bldr, len(cols))
	for i := range offsets {
		bldr.PrependUOffsetT(offsets[len(cols)-i-1])
	}
	return bldr.EndVector(len(cols))
}

type mockCRS struct {
	org         *string
	code        int32
	name        *string
	description *string
	wkt         *string
	codeString  *string
}

func mockCRSFromFlatBufferTable(crs *flat.Crs) (mc mockCRS) {
	if crs.Org() != nil {
		mc.org = stringPtr(string(crs.Org()))
	}
	mc.code = crs.Code()
	if crs.Name() != nil {
		mc.name = stringPtr(string(crs.Name()))
	}
	if crs.Description() != nil {
		mc.description = stringPtr(string(crs.Description()))
	}
	if crs.Wkt() != nil {
		mc.wkt = stringPtr(string(crs.Wkt()))
	}
	if crs.CodeString() != nil {
		mc.codeString = stringPtr(string(crs.CodeString()))
	}
	return
}

func (m *mockCRS) build(bldr *flatbuffers.Builder) flatbuffers.UOffsetT {
	func() {
		if m.org != nil {
			offset := bldr.CreateString(*m.org)
			defer flat.CrsAddOrg(bldr, offset)
		}
		if m.code != 0 {
			defer flat.CrsAddCode(bldr, m.code)
		}
		if m.name != nil {
			offset := bldr.CreateString(*m.name)
			defer flat.CrsAddName(bldr, offset)
		}
		if m.description != nil {
			offset := bldr.CreateString(*m.description)
			defer flat.CrsAddDescription(bldr, offset)
		}
		if m.wkt != nil {
			offset := bldr.CreateString(*m.wkt)
			defer flat.CrsAddWkt(bldr, offset)
		}
		if m.codeString != nil {
			offset := bldr.CreateString(*m.codeString)
			defer flat.CrsAddCodeString(bldr, offset)
		}
		flat.CrsStart(bldr)
	}()
	return flat.CrsEnd(bldr)
}

type mockHeader struct {
	name          *string
	envelope      []float64
	geometryType  flat.GeometryType
	hasZ          bool
	hasM          bool
	hasT          bool
	hasTM         bool
	columns       []mockColumn
	featuresCount uint64
	indexNodeSize *uint16
	crs           *mockCRS
	title         *string
	description   *string
	metadata      *string
}

func mockHeaderFromFlatBufferTable(hdr *flat.Header) (mh mockHeader) {
	if hdr.Name() != nil {
		mh.name = stringPtr(string(hdr.Name()))
	}
	if hdr.EnvelopeLength() > 0 {
		mh.envelope = make([]float64, hdr.EnvelopeLength())
		for i := range mh.envelope {
			mh.envelope[i] = hdr.Envelope(i)
		}
	}
	mh.geometryType = hdr.GeometryType()
	mh.hasZ = hdr.HasZ()
	mh.hasM = hdr.HasM()
	mh.hasT = hdr.HasT()
	mh.hasTM = hdr.HasTm()
	if hdr.ColumnsLength() > 0 {
		mh.columns = make([]mockColumn, hdr.ColumnsLength())
		for i := range mh.columns {
			var flatColumn flat.Column
			hdr.Columns(&flatColumn, i)
			mh.columns[i] = mockColumnFromFlatBufferTable(&flatColumn)
		}
	}
	mh.featuresCount = hdr.FeaturesCount()
	mh.indexNodeSize = uint16Ptr(hdr.IndexNodeSize())
	var flatCRS flat.Crs
	if hdr.Crs(&flatCRS) != nil {
		crs := mockCRSFromFlatBufferTable(&flatCRS)
		mh.crs = &crs
	}
	if hdr.Title() != nil {
		mh.title = stringPtr(string(hdr.Title()))
	}
	if hdr.Description() != nil {
		mh.description = stringPtr(string(hdr.Description()))
	}
	if hdr.Metadata() != nil {
		mh.metadata = stringPtr(string(hdr.Metadata()))
	}
	return
}

func (mh *mockHeader) build(bldr *flatbuffers.Builder) flatbuffers.UOffsetT {
	func() {
		if mh.name != nil {
			offset := bldr.CreateString(*mh.name)
			defer flat.HeaderAddName(bldr, offset)
		}
		if len(mh.envelope) > 0 {
			n := len(mh.envelope)
			flat.HeaderStartEnvelopeVector(bldr, n)
			for i := n - 1; i >= 0; i-- {
				bldr.PrependFloat64(mh.envelope[i])
			}
			offset := bldr.EndVector(n)
			defer flat.HeaderAddEnvelope(bldr, offset)
		}
		if mh.geometryType != 0 {
			defer flat.HeaderAddGeometryType(bldr, mh.geometryType)
		}
		if mh.hasZ {
			defer flat.HeaderAddHasZ(bldr, true)
		}
		if mh.hasM {
			defer flat.HeaderAddHasM(bldr, true)
		}
		if mh.hasT {
			defer flat.HeaderAddHasT(bldr, true)
		}
		if mh.hasTM {
			defer flat.HeaderAddHasTm(bldr, true)
		}
		if mh.columns != nil {
			offset := buildMockColumns(bldr, mh.columns, flat.HeaderStartColumnsVector)
			defer flat.HeaderAddColumns(bldr, offset)
		}
		if mh.featuresCount > 0 {
			defer flat.HeaderAddFeaturesCount(bldr, mh.featuresCount)
		}
		if mh.indexNodeSize != nil {
			defer flat.HeaderAddIndexNodeSize(bldr, *mh.indexNodeSize)
		}
		if mh.crs != nil {
			offset := mh.crs.build(bldr)
			defer flat.HeaderAddCrs(bldr, offset)
		}
		if mh.title != nil {
			offset := bldr.CreateString(*mh.title)
			defer flat.HeaderAddTitle(bldr, offset)
		}
		if mh.description != nil {
			offset := bldr.CreateString(*mh.description)
			defer flat.HeaderAddDescription(bldr, offset)
		}
		if mh.metadata != nil {
			offset := bldr.CreateString(*mh.metadata)
			defer flat.HeaderAddMetadata(bldr, offset)
		}
		flat.HeaderStart(bldr)
	}()
	return flat.HeaderEnd(bldr)
}

func (mh *mockHeader) buildAsBytes() []byte {
	bldr := flatbuffers.NewBuilder(2048)
	hdr := mh.build(bldr)
	flat.FinishSizePrefixedHeaderBuffer(bldr, hdr)
	return bldr.FinishedBytes()
}

type mockGeometry struct {
	ends         []uint32
	xy           []float64
	z            []float64
	m            []float64
	t            []float64
	tm           []uint64
	geometryType flat.GeometryType
	parts        []mockGeometry
}

func (mg *mockGeometry) build(bldr *flatbuffers.Builder) flatbuffers.UOffsetT {
	func() {
		if len(mg.ends) > 0 {
			bldr.StartVector(flatbuffers.SizeUint32, len(mg.ends), flatbuffers.SizeUint32)
			for i := len(mg.ends) - 1; i >= 0; i-- {
				bldr.PrependUint32(mg.ends[i])
			}
			offset := bldr.EndVector(len(mg.ends))
			defer flat.GeometryAddEnds(bldr, offset)
		}
		if len(mg.xy) > 0 {
			offset := buildDoubleVector(bldr, mg.xy)
			defer flat.GeometryAddXy(bldr, offset)
		}
		if len(mg.z) > 0 {
			offset := buildDoubleVector(bldr, mg.z)
			defer flat.GeometryAddZ(bldr, offset)
		}
		if len(mg.m) > 0 {
			offset := buildDoubleVector(bldr, mg.m)
			defer flat.GeometryAddM(bldr, offset)
		}
		if len(mg.t) > 0 {
			offset := buildDoubleVector(bldr, mg.t)
			defer flat.GeometryAddXy(bldr, offset)
		}
		if len(mg.tm) > 0 {
			bldr.StartVector(flatbuffers.SizeUint64, len(mg.tm), flatbuffers.SizeUint64)
			for i := len(mg.tm) - 1; i >= 0; i-- {
				bldr.PrependUint64(mg.tm[i])
			}
			offset := bldr.EndVector(len(mg.tm))
			defer flat.GeometryAddTm(bldr, offset)
		}
		defer flat.GeometryAddType(bldr, mg.geometryType)
		if len(mg.parts) > 0 {
			offsets := make([]flatbuffers.UOffsetT, len(mg.parts))
			for i := range mg.parts {
				offsets[i] = mg.parts[i].build(bldr)
			}
			offset := bldr.CreateVectorOfTables(offsets)
			defer flat.GeometryAddParts(bldr, offset)
		}
		flat.GeometryStart(bldr)
	}()
	return flat.GeometryEnd(bldr)
}

type mockFeature struct {
	geometry   *mockGeometry
	properties []byte
	columns    []mockColumn
}

func (mf *mockFeature) build(bldr *flatbuffers.Builder) flatbuffers.UOffsetT {
	func() {
		if mf.geometry != nil {
			offset := mf.geometry.build(bldr)
			defer flat.FeatureAddGeometry(bldr, offset)
		}
		if len(mf.properties) > 0 {
			offset := bldr.CreateByteVector(mf.properties)
			defer flat.FeatureAddProperties(bldr, offset)
		}
		if len(mf.columns) < 0 {
			offset := buildMockColumns(bldr, mf.columns, flat.FeatureStartColumnsVector)
			defer flat.FeatureAddColumns(bldr, offset)
		}
		flat.FeatureStart(bldr)
	}()
	return flat.FeatureEnd(bldr)
}

func (mf *mockFeature) buildAsBytes() []byte {
	bldr := flatbuffers.NewBuilder(2048)
	hdr := mf.build(bldr)
	flat.FinishSizePrefixedFeatureBuffer(bldr, hdr)
	return bldr.FinishedBytes()
}

type mockFile struct {
	name        string
	headerBytes []byte
	header      *mockHeader
	indexOffset int64
	indexBytes  []byte
	indexFunc   func() (*packedrtree.PackedRTree, error)
	dataOffsets []int64
	dataBytes   [][]byte
	data        []mockFeature
	once        sync.Once
	allBytes    []byte
}

func (mf *mockFile) init(t *testing.T) {
	mf.once.Do(func() {
		var b bytes.Buffer

		// Magic number
		_, _ = b.Write(magic[:])

		// Header
		if mf.headerBytes != nil && mf.header != nil {
			t.Fatalf("headerBytes and header are both set for mockFile %q", mf.name)
		} else if mf.header != nil {
			mf.headerBytes = mf.header.buildAsBytes()
		}
		_, _ = b.Write(mf.headerBytes)

		// Index
		mf.indexOffset = int64(magicLen) + int64(len(mf.headerBytes))
		if mf.indexBytes != nil && mf.indexFunc != nil {
			t.Fatalf("indexBytes and indexFunc are both set for mockFile %q", mf.name)
		} else if mf.indexFunc != nil {
			var c bytes.Buffer
			index, err := mf.indexFunc()
			require.NoError(t, err, "failed to create index for mockFile %q", mf.name)
			_, _ = index.Marshal(&c)
			mf.indexBytes = c.Bytes()
		}
		_, _ = b.Write(mf.indexBytes)

		// Features
		if mf.dataBytes != nil && mf.data != nil {
			t.Fatalf("dataBytes and data are both set for mockFile %q", mf.name)
		} else if mf.data != nil {
			mf.dataBytes = make([][]byte, len(mf.data))
			for i := range mf.data {
				mf.dataBytes[i] = mf.data[i].buildAsBytes()
			}
		}
		if len(mf.dataBytes) == 0 {
			mf.dataOffsets = []int64{mf.indexOffset + int64(len(mf.indexBytes))}
		} else {
			mf.dataOffsets = make([]int64, len(mf.dataBytes))
			mf.dataOffsets[0] = mf.indexOffset + int64(len(mf.indexBytes))
			_, _ = b.Write(mf.dataBytes[0])
			for i := 1; i < len(mf.dataBytes); i++ {
				mf.dataOffsets[i] = mf.dataOffsets[i-1] + int64(len(mf.dataBytes[i-1]))
				_, _ = b.Write(mf.dataBytes[i])
			}
		}

		// Finished the whole file
		mf.allBytes = b.Bytes()
	})
}

var mockFiles = []mockFile{
	{
		name:   "empty",
		header: &mockHeader{},
	},
	{
		name:        "truncated_header",
		headerBytes: make([]byte, 0),
	},
	{
		name: "truncated_index",
		header: &mockHeader{
			featuresCount: 1,
			indexNodeSize: uint16Ptr(64),
		},
		indexBytes: make([]byte, 0),
	},
	{
		name: "truncated_data",
		header: &mockHeader{
			featuresCount: 1,
			indexNodeSize: uint16Ptr(2),
		},
		indexFunc: func() (*packedrtree.PackedRTree, error) {
			return packedrtree.New([]packedrtree.Ref{
				{
					Box:    packedrtree.Box{XMin: 0, YMin: 0, XMax: 0, YMax: 0},
					Offset: 0,
				},
			}, 2)
		},
	},
	{
		name: "feature_length_too_small",
		header: &mockHeader{
			indexNodeSize: uint16Ptr(0),
		},
		dataBytes: [][]byte{
			make([]byte, flatbuffers.SizeUint32), // Indicates a feature of length zero.
		},
	},
	{
		name: "index_size_overflow",
		header: &mockHeader{
			featuresCount: math.MaxInt64,
			indexNodeSize: uint16Ptr(1024),
		},
	},
	{
		name: "one_feature_no_index",
		header: &mockHeader{
			featuresCount: 1,
			indexNodeSize: uint16Ptr(0),
		},
		data: []mockFeature{
			{
				geometry: &mockGeometry{
					xy:           []float64{-1, -1, 0, 0},
					geometryType: flat.GeometryTypeLineString,
				},
			},
		},
	},
	{
		name: "one_feature_no_index_unknown_count",
		header: &mockHeader{
			featuresCount: 0,
			indexNodeSize: uint16Ptr(0),
		},
		data: []mockFeature{
			{
				geometry: &mockGeometry{
					xy:           []float64{-1, -1, 0, 0},
					geometryType: flat.GeometryTypeLineString,
				},
			},
		},
	},
	{
		name: "one_feature_with_index",
		header: &mockHeader{
			featuresCount: 1,
			indexNodeSize: uint16Ptr(100),
		},
		indexFunc: func() (*packedrtree.PackedRTree, error) {
			return packedrtree.New([]packedrtree.Ref{
				{
					Box:    packedrtree.Box{XMin: -1, YMin: -1, XMax: 0, YMax: 0},
					Offset: 0,
				},
			}, 2)
		},
		data: []mockFeature{
			{
				geometry: &mockGeometry{
					xy:           []float64{-1, -1, 0, 0},
					geometryType: flat.GeometryTypeLineString,
				},
			},
		},
	},
	{},
	{
		// This test case has four point features, one in each quadrant,
		// with an overall bounding box of [-1, -1, 1, 1].
		name: "four_points_in_quadrants",
		header: &mockHeader{
			featuresCount: 4,
			indexNodeSize: uint16Ptr(2),
		},
		indexFunc: func() (*packedrtree.PackedRTree, error) {
			// The below list of refs is already Hilbert-sorted. Note
			// that the four point features are each 80 bytes in total,
			// hence the offsets.
			refs := []packedrtree.Ref{
				{
					Box:    packedrtree.Box{XMin: -1, YMin: -1, XMax: -1, YMax: -1},
					Offset: 0,
				},
				{
					Box:    packedrtree.Box{XMin: -1, YMin: 1, XMax: -1, YMax: 1},
					Offset: 80,
				},
				{
					Box:    packedrtree.Box{XMin: 1, YMin: 1, XMax: 1, YMax: 1},
					Offset: 160,
				},
				{
					Box:    packedrtree.Box{XMin: 1, YMin: -1, XMax: 1, YMax: -1},
					Offset: 240,
				},
			}
			return packedrtree.New(refs, 2)
		},
		data: []mockFeature{
			{
				geometry: &mockGeometry{
					xy:           []float64{-1, -1},
					geometryType: flat.GeometryTypePoint,
				},
			},
			{
				geometry: &mockGeometry{
					xy:           []float64{-1, 1},
					geometryType: flat.GeometryTypePoint,
				},
			},
			{
				geometry: &mockGeometry{
					xy:           []float64{1, 1},
					geometryType: flat.GeometryTypePoint,
				},
			},
			{
				geometry: &mockGeometry{
					xy:           []float64{1, -1},
					geometryType: flat.GeometryTypePoint,
				},
			},
		},
	},
}

type mockDataReadError struct {
	pos int64
	err error
}

type mockDataBytesReader struct {
	t   *testing.T
	mf  *mockFile
	rs  io.ReadSeeker
	pos int64
	err *mockDataReadError
}

func newMockDataBytesReader(t *testing.T, mf *mockFile, err *mockDataReadError) *mockDataBytesReader {
	require.NotNil(t, mf.allBytes, "mockFile not initialized: %q", mf.name)
	r := &mockDataBytesReader{
		t:   t,
		mf:  mf,
		rs:  bytes.NewReader(mf.allBytes),
		err: err,
	}
	return r
}

func (m *mockDataBytesReader) Read(p []byte) (n int, err error) {
	if m.err != nil {
		end := m.pos + int64(len(p))
		if end > m.err.pos {
			x := m.err.pos - m.pos
			y, err := m.rs.Read(p[:x])
			m.pos += int64(y)
			if err != nil {
				return y, err
			} else if int64(y) < x {
				return y, nil
			} else {
				err = m.err.err
				m.err = nil
				return int(x), err
			}
		}
	}
	n, err = m.rs.Read(p)
	m.pos += int64(n)
	return
}

func (m *mockDataBytesReader) verify() {
	if m.err != nil {
		m.t.Fatalf("expected to provide error %s on Read of byte position %d, but it was never read (current position is %d)", m.err.err, m.err.pos, m.pos)
	}
}

type mockDataSeekError struct {
	offset int64
	whence int
	time   int
	err    error
}

type mockDataBytesReadSeeker struct {
	mockDataBytesReader
	err *mockDataSeekError
}

func newMockDataBytesReadSeeker(t *testing.T, mf *mockFile, readErr *mockDataReadError, seekErr *mockDataSeekError) *mockDataBytesReadSeeker {
	return &mockDataBytesReadSeeker{
		mockDataBytesReader: *newMockDataBytesReader(t, mf, readErr),
		err:                 seekErr,
	}
}

func (m *mockDataBytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if m.err != nil && m.err.offset == offset && m.err.whence == whence {
		if m.err.time == 0 {
			err := m.err.err
			m.err = nil
			return 0, err
		} else {
			m.err.time--
		}
	}
	n, err := m.rs.Seek(offset, whence)
	m.pos = n
	return n, err
}

func (m *mockDataBytesReadSeeker) verify() {
	m.mockDataBytesReader.verify()
	if m.err != nil {
		m.t.Fatalf("expected to provide error %s on Seek to offset %d from (whence) %d, but the seek was never requested", m.err.err, m.err.offset, m.err.whence)
	}
}

func mockDataRunTest(t *testing.T, f func(t *testing.T, mf *mockFile), name string) {
	t.Run(name, func(t *testing.T) {
		mf := func() *mockFile {
			for i := range mockFiles {
				if mockFiles[i].name == name {
					return &mockFiles[i]
				}
			}
			t.Fatalf("no such mock FlatGeobuf file: %q", name)
			return nil
		}()
		mf.init(t)
		f(t, mf)
	})
}

func stringPtr(s string) *string {
	return &s
}

func uint16Ptr(u uint16) *uint16 {
	return &u
}

func buildDoubleVector(bldr *flatbuffers.Builder, v []float64) flatbuffers.UOffsetT {
	bldr.StartVector(flatbuffers.SizeFloat64, len(v), flatbuffers.SizeFloat64)
	for i := len(v) - 1; i >= 0; i-- {
		bldr.PrependFloat64(v[i])
	}
	return bldr.EndVector(len(v))
}
