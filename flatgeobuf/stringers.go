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
		var needComma bool
		needComma = stringBytes(&b, needComma, "Name", hdr.Name()) || needComma
		needComma = stringEnvelope(&b, needComma, hdr) || needComma
		stringStr(&b, needComma, "Type", hdr.GeometryType().String())
		stringHeaderFlags(&b, hdr.HasZ(), hdr.HasM(), hdr.HasT(), hdr.HasTm())
		stringColumns(&b, hdr)
		numFeatures := hdr.FeaturesCount()
		if numFeatures > 0 {
			stringUint64(&b, true, "Features", hdr.FeaturesCount())
		} else {
			stringStr(&b, true, "Features", "Unknown")
		}
		nodeSize := hdr.IndexNodeSize()
		if nodeSize > 0 {
			stringUint64(&b, true, "NodeSize", uint64(nodeSize))
		} else {
			_, _ = fmt.Fprint(&b, ",No Index")
		}
		var crs flat.Crs
		stringKey(&b, true, "CRS")
		if hdr.Crs(&crs) != nil {
			b.WriteByte('{')
			needComma = stringBytes(&b, false, "Org", crs.Org())
			stringInt64(&b, needComma, "Code", int64(crs.Code()))
			stringBytes(&b, true, "Name", crs.Name())
			stringBytes(&b, true, "Description", crs.Description())
			wkt := crs.Wkt()
			stringKey(&b, true, "WKT")
			if wkt == nil {
				b.WriteString("<nil>")
			} else {
				_, _ = fmt.Fprintf(&b, "%d bytes", len(wkt))
			}
			stringBytes(&b, true, "CodeString", crs.CodeString())
			b.WriteByte('}')
		} else {
			b.WriteString("<nil>")
		}
		stringBytes(&b, true, "Title", hdr.Title())
		stringBytes(&b, true, "Desc", hdr.Description())
		stringBytes(&b, true, "Meta", hdr.Metadata())
		return nil
	}); err != nil {
		return "Header{error: " + err.Error() + "}"
	}
	b.WriteByte('}')
	return b.String()
}

func stringKey(b *strings.Builder, needComma bool, key string) {
	if needComma {
		b.WriteByte(',')
	}
	b.WriteString(key)
	b.WriteByte(':')
}

func stringBytes(b *strings.Builder, needComma bool, key string, value []byte) bool {
	if value == nil {
		return false
	}
	stringKey(b, needComma, key)
	b.Write(value)
	return true
}

func stringStr(b *strings.Builder, needComma bool, key string, value string) {
	stringKey(b, needComma, key)
	b.WriteString(value)
}

func stringInt64(b *strings.Builder, needComma bool, key string, value int64) {
	stringKey(b, needComma, key)
	_, _ = fmt.Fprintf(b, "%d", value)
}
func stringUint64(b *strings.Builder, needComma bool, key string, value uint64) {
	stringKey(b, needComma, key)
	_, _ = fmt.Fprintf(b, "%d", value)
}

func stringEnvelope(b *strings.Builder, needComma bool, hdr *flat.Header) bool {
	n := hdr.EnvelopeLength()
	if n < 1 {
		return false
	}
	stringKey(b, needComma, "Envelope")
	b.WriteByte('[')
	_, _ = fmt.Fprintf(b, "%.8g", hdr.Envelope(0))
	for i := 1; i < n; i++ {
		_, _ = fmt.Fprintf(b, ",%.8g", hdr.Envelope(i))
	}
	b.WriteByte(']')
	return true
}

func stringHeaderFlags(b *strings.Builder, z, m, t, tm bool) {
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

func stringColumns(b *strings.Builder, s Schema) {
	if n := s.ColumnsLength(); n > 0 {
		stringInt64(b, true, "Columns", int64(n))
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
		return "Feature{error: geometry: " + err.Error() + "}"
	}
	b.WriteString(",Properties:{")
	ss := make([]Schema, 1, 2)
	ss[0] = f
	if s != nil {
		ss = append(ss, s)
	}
	if err := stringProps(f, &b, ss); err != nil {
		return "Feature{error: properties: " + err.Error() + "}"
	}
	b.WriteByte('}')
	stringColumns(&b, f)
	b.WriteByte('}')
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
			stringGeomCounts(b, g.EndsLength(), g.ZLength(), g.MLength(), g.TLength(), g.TmLength(), g.PartsLength())
			b.WriteByte('}')
		} else {
			b.WriteString("<nil>")
		}
		return nil
	})
}

func stringGeomCounts(b *strings.Builder, ends, z, m, t, tm, parts int) {
	if ends > 0 || z > 0 || m > 0 || t > 0 || tm > 0 || parts > 0 {
		b.WriteByte(',')
		var numPrinted int
		count := func(name string, value int) {
			if value > 0 {
				stringInt64(b, numPrinted > 0, name, int64(value))
				numPrinted++
			}
		}
		count("Ends", ends)
		count("Z", z)
		count("M", m)
		count("T", t)
		count("TM", tm)
		count("Parts", parts)
	}
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
	for i := 0; i+1 < n; i += 2 {
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
