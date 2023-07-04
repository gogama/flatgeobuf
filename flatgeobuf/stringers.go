// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"

	"github.com/gogama/flatgeobuf/packedrtree"
)

// HeaderString returns a string summarizing the Header fields. The
// returned value is a summary and not meant to be exhaustive.
func HeaderString(hdr *flat.Header) string {
	var b strings.Builder
	b.WriteString("Header{")
	if err := safeFlatBuffersInteraction(func() error {
		stringBytes(&b, "Name", hdr.Name())
		stringEnvelope(&b, hdr)
		stringStr(&b, ",Type", hdr.GeometryType().String())
		stringFlags(&b, hdr.HasZ(), hdr.HasM(), hdr.HasT(), hdr.HasTm())
		stringInt64(&b, ",NumColumns", int64(hdr.ColumnsLength()))
		numFeatures := hdr.FeaturesCount()
		if numFeatures > 0 {
			stringUint64(&b, ",NumFeatures", hdr.FeaturesCount())
		} else {
			stringStr(&b, ",NumFeatures", "UNKNOWN")
		}
		nodeSize := hdr.IndexNodeSize()
		if nodeSize > 0 {
			stringUint64(&b, ",NodeSize", uint64(nodeSize))
		} else {
			fmt.Fprint(&b, ",NO INDEX")
		}
		var crs flat.Crs
		stringKey(&b, ",CRS")
		if hdr.Crs(&crs) != nil {
			b.WriteByte('{')
			stringBytes(&b, "Org", crs.Org())
			stringInt64(&b, ",Code", int64(crs.Code()))
			stringBytes(&b, ",Name", crs.Name())
			wkt := crs.Wkt()
			stringKey(&b, ",WKT")
			if wkt == nil {
				b.WriteString("<nil>")
			} else {
				fmt.Fprintf(&b, "%d bytes", len(wkt))
			}
			stringBytes(&b, ",CodeString", crs.CodeString())
			b.WriteByte('}')
		} else {
			b.WriteString("<nil>")
		}
		stringBytes(&b, ",Title", hdr.Title())
		stringBytes(&b, ",Desc", hdr.Description())
		stringBytes(&b, ",Meta", hdr.Metadata())
		return nil
	}); err != nil {
		return "error: " + err.Error()
	}
	b.WriteByte('}')
	return b.String()
}

func stringKey(b *strings.Builder, key string) {
	b.WriteString(key)
	b.WriteByte(':')
}

func stringBytes(b *strings.Builder, key string, value []byte) {
	if value != nil {
		stringKey(b, key)
		b.Write(value)
	}
}

func stringStr(b *strings.Builder, key string, value string) {
	stringKey(b, key)
	b.WriteString(value)
}

func stringInt64(b *strings.Builder, key string, value int64) {
	stringKey(b, key)
	fmt.Fprintf(b, "%d", value)
}
func stringUint64(b *strings.Builder, key string, value uint64) {
	stringKey(b, key)
	fmt.Fprintf(b, "%d", value)
}

func stringEnvelope(b *strings.Builder, hdr *flat.Header) {
	n := hdr.EnvelopeLength()
	if n > 0 {
		stringKey(b, ",Envelope")
		b.WriteByte('[')
		fmt.Fprintf(b, "%.8g", hdr.Envelope(0))
		for i := 1; i < n; i++ {
			fmt.Fprintf(b, ",%.8g", hdr.Envelope(i))
		}
		b.WriteByte(']')
	}
}

func stringFlags(b *strings.Builder, z, m, t, tm bool) {
	if z || m || t || tm {
		b.WriteByte(',')
		var numPrinted int
		flag := func(name string, value bool) {
			if value {
				if numPrinted > 0 {
					b.WriteByte('|')
				}
				b.WriteString(name)
				numPrinted++
			}
		}
		flag("Z", z)
		flag("M", m)
		flag("T", t)
		flag("TM", tm)
	}
}

// FeatureString returns a string summarizing the Feature. The returned
// value is a summary and not meant to be exhaustive.
//
// Property column names are taken from the Feature's column schema, if
// it has one. If not, they are taken from the supplied Schema parameter
// if it is not nil. The supplied Schema parameter will typically be
// the *flat.Header from the feature's FlatGeobuf file.
func FeatureString(f *flat.Feature, s Schema) string {
	var b strings.Builder
	b.WriteString("Feature{Geometry:")
	if err := stringGeom(f, &b); err != nil {
		return "error: geometry: " + err.Error()
	}
	b.WriteString(",Properties:{")
	ss := make([]Schema, 1, 2)
	ss[0] = f
	if s != nil {
		ss = append(ss, s)
	}
	if err := stringProps(f, &b, ss); err != nil {
		return "error: properties: " + err.Error()
	}
	b.WriteString("}}")
	return b.String()
}

func stringGeom(f *flat.Feature, b *strings.Builder) error {
	return safeFlatBuffersInteraction(func() error {
		var g flat.Geometry
		if f.Geometry(&g) != nil {
			b.WriteString("{Type:")
			b.WriteString(g.Type().String())
			b.WriteString(",Bounds:")
			bounds := packedrtree.EmptyBox
			geomBounds(&g, &bounds)
			if bounds == packedrtree.EmptyBox {
				b.WriteString("<nil>")
			} else {
				b.WriteString(bounds.String())
			}
			b.WriteByte('}')
		} else {
			b.WriteString("<nil>")
		}
		return nil
	})
}

func stringProps(f *flat.Feature, b *strings.Builder, s []Schema) error {
	return safeFlatBuffersInteraction(func() error {
		// Pick the lowest indexed schema which has at least one
		// column.
		schema := s[0]
		n := schema.ColumnsLength()
		for i := 1; i < len(s) && n == 0; i++ {
			if n2 := s[i].ColumnsLength(); n2 > 0 {
				schema = s[i]
				n = n2
			}
		}
		// Generate the properties using the schema we picked.
		r := NewPropReader(bytes.NewReader(f.PropertiesBytes()))
		var vals []PropValue
		var err error
		if vals, err = r.ReadSchema(schema); err != nil {
			return err
		}
		printFunc := func(vv *PropValue, i int) {
			if len(vv.Col.Name()) > 0 {
				b.Write(vv.Col.Name())
			} else {
				_, _ = fmt.Fprintf(b, "[%d]", i)
			}
			b.WriteByte(':')
			_, _ = fmt.Fprint(b, vv.Value)

		}
		if len(vals) > 0 {
			printFunc(&vals[0], 0)
			for i := 1; i < len(vals); i++ {
				b.WriteByte(',')
				printFunc(&vals[i], i)
			}
		}
		return nil
	})
}

func geomBounds(g *flat.Geometry, b *packedrtree.Box) {
	n := g.XyLength()
	for i := 0; i < n; i += 2 {
		b.ExpandXY(g.Xy(i+0), g.Xy(i+1))
	}
	n = g.PartsLength()
	for i := 0; i < n; i++ {
		var h flat.Geometry
		if g.Parts(&h, i) {
			geomBounds(&h, b)
		}
	}
}
