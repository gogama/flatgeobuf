package packedrtree

import "math"

type Box struct {
	XMin float64
	YMin float64
	XMax float64
	YMax float64
}

// TODO: EmptyBox is probably a better name.
var Null = Box{
	XMin: math.Inf(1),
	YMin: math.Inf(1),
	XMax: math.Inf(-1),
	YMax: math.Inf(-1),
}

func (b *Box) Width() float64 {
	return b.XMax - b.XMin
}

func (b *Box) Height() float64 {
	return b.YMax - b.YMin
}

func (b *Box) midX() float64 {
	return (b.XMin + b.XMax) / 2
}

func (b *Box) midY() float64 {
	return (b.YMin + b.YMin) / 2
}

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
		return true
	}
	if b.YMax < o.YMin {
		return true
	}
	if b.XMin > o.XMax {
		return true
	}
	if b.YMin > o.YMax {
		return true
	}
	return false
}
