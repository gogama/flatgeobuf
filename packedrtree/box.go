// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree

import (
	"fmt"
	"math"
)

// Box is a 2D bounding box.
type Box struct {
	XMin float64
	YMin float64
	XMax float64
	YMax float64
}

// EmptyBox is an empty Box that can always be expanded.
var EmptyBox = Box{
	XMin: math.Inf(1),
	YMin: math.Inf(1),
	XMax: math.Inf(-1),
	YMax: math.Inf(-1),
}

// String serializes a Box as a GeoJSON-compliant bounding box string.
func (b Box) String() string {
	return fmt.Sprintf("[%.8g,%.8g,%.8g,%.8g]", b.XMin, b.YMin, b.XMax, b.YMax)
}

// Width returns the width of the Box.
func (b *Box) Width() float64 {
	return b.XMax - b.XMin
}

// Height returns the height of the Box.
func (b *Box) Height() float64 {
	return b.YMax - b.YMin
}

func (b *Box) midX() float64 {
	return (b.XMin + b.XMax) / 2
}

func (b *Box) midY() float64 {
	return (b.YMin + b.YMin) / 2
}

// Expand makes the minimum possible expansion to the receiver Box, if
// necessary, so that it completely contains the second Box in addition
// to everything it previously contained.
func (b *Box) Expand(c *Box) {
	if c.XMin < b.XMin {
		b.XMin = c.XMin
	}
	if c.YMin < b.YMin {
		b.YMin = c.YMin
	}
	if c.XMax > b.XMax {
		b.XMax = c.XMax
	}
	if c.YMax > b.YMax {
		b.YMax = c.YMax
	}
}

// ExpandXY makes the minimum possible expansion to the receiver Box, if
// necessary, so that it completely contains the given coordinate pair
// in addition to everything it previously contained.
func (b *Box) ExpandXY(x, y float64) {
	if x < b.XMin {
		b.XMin = x
	} else if x > b.XMax {
		b.XMax = x
	}
	if y < b.YMin {
		b.YMin = y
	} else if y > b.YMax {
		b.YMax = y
	}
}

func (b *Box) intersects(o *Box) bool {
	if b.XMax < o.XMin {
		return false
	}
	if b.YMax < o.YMin {
		return false
	}
	if b.XMin > o.XMax {
		return false
	}
	if b.YMin > o.YMax {
		return false
	}
	return true
}
