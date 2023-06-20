// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package flatgeobuf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/gogama/flatgeobuf/packedrtree"
)

func (f *Feature) String() string {
	return f.string(f)
}

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
