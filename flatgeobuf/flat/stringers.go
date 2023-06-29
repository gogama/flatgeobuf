// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flat

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gogama/flatgeobuf/packedrtree"
)

// String returns a string summarizing the Header fields. The returned
// value is a summary and not meant to be exhaustive.
func (h *Header) String() string {
	var b strings.Builder
	b.WriteString("Header{")
	if err := safeFlatBuffersInteraction(func() error {
		stringBytes(&b, "Name", h.Name())
		stringEnvelope(&b, h)
		stringStr(&b, ",Type", h.GeometryType().String())
		stringFlags(&b, h.HasZ(), h.HasM(), h.HasT(), h.HasTm())
		stringInt64(&b, ",NumColumns", int64(h.ColumnsLength()))
		numFeatures := h.FeaturesCount()
		if numFeatures > 0 {
			stringUint64(&b, ",NumFeatures", h.FeaturesCount())
		} else {
			stringStr(&b, ",NumFeatures", "UNKNOWN")
		}
		nodeSize := h.IndexNodeSize()
		if nodeSize > 0 {
			stringUint64(&b, ",NodeSize", uint64(nodeSize))
		} else {
			fmt.Fprint(&b, ",NO INDEX")
		}
		var crs Crs
		stringKey(&b, ",CRS")
		if h.Crs(&crs) != nil {
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
		stringBytes(&b, ",Title", h.Title())
		stringBytes(&b, ",Desc", h.Description())
		stringBytes(&b, ",Meta", h.Metadata())
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

func stringEnvelope(b *strings.Builder, h *Header) {
	n := h.EnvelopeLength()
	if n > 0 {
		stringKey(b, ",Envelope")
		b.WriteByte('[')
		fmt.Fprintf(b, "%.8g", h.Envelope(0))
		for i := 1; i < n; i++ {
			fmt.Fprintf(b, ",%.8g", h.Envelope(i))
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

// String returns a string summarizing the Feature. The returned value
// is a summary and not meant to be exhaustive.
//
// If the column schema is external to the Feature (i.e. it comes from
// the file Header), the method StringSchema should be used. This method
// will return a harmless, but useless, string containing an error
// message.
func (f *Feature) String() string {
	return f.string(f)
}

// StringSchema returns a string summarizing the Feature. The returned
// value is a summary and not meant to be exhaustive.
//
// Property column names are taken from the Feature's column schema, if
// it has one, and the supplied Schema parameter otherwise.
func (f *Feature) StringSchema(s Schema) string {
	return f.string(f, s)
}

func (f *Feature) string(s ...Schema) string {
	var b strings.Builder
	b.WriteString("Feature{Geometry:")
	if err := f.stringGeom(&b); err != nil {
		return "error: geometry: " + err.Error()
	}
	b.WriteString(",Properties:{")
	if err := f.stringProps(&b, s...); err != nil {
		return "error: properties: " + err.Error()
	}
	b.WriteString("}}")
	return b.String()
}

func (f *Feature) stringGeom(b *strings.Builder) error {
	return safeFlatBuffersInteraction(func() error {
		var g Geometry
		if f.Geometry(&g) != nil {
			b.WriteString("{Type:")
			b.WriteString(g.Type().String())
			b.WriteString(",Bounds:")
			bounds := packedrtree.EmptyBox
			g.bounds(&bounds)
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

func (f *Feature) stringProps(b *strings.Builder, s ...Schema) error {
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

func (g *Geometry) bounds(b *packedrtree.Box) {
	n := g.XyLength()
	for i := 0; i < n; i += 2 {
		b.ExpandXY(g.Xy(i+0), g.Xy(i+1))
	}
	n = g.PartsLength()
	for i := 0; i < n; i++ {
		var h Geometry
		if g.Parts(&h, i) {
			h.bounds(b)
		}
	}
}
