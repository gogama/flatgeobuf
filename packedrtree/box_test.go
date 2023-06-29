// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBox_String(t *testing.T) {
	testCases := []struct {
		name     string
		input    Box
		expected string
	}{
		{"Zero", Box{}, "[0,0,0,0]"},
		{"Integers", Box{-1, 2, -3, 4}, "[-1,2,-3,4]"},
		{"Exact", Box{-100.5, -200.25, 1234.125, 5678.0625}, "[-100.5,-200.25,1234.125,5678.0625]"},
		{"Rounded", Box{-100000.0625, 123.015625, 99.0078125, -2.001953125}, "[-100000.06,123.01562,99.007812,-2.0019531]"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.input.String()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestBox_Width(t *testing.T) {
	testCases := []struct {
		name     string
		input    Box
		expected float64
	}{
		{"Zero", Box{}, 0},
		{"One", Box{0, 0, 1, 0}, 1},
		{"Two", Box{-1, 0, 1, 0}, 2},
		{"Empty", EmptyBox, math.Inf(-1)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.input.Width()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestBox_Height(t *testing.T) {
	testCases := []struct {
		name     string
		input    Box
		expected float64
	}{
		{"Zero", Box{}, 0},
		{"One", Box{0, 0, 0, 1}, 1},
		{"Two", Box{0, -1, 0, 1}, 2},
		{"Empty", EmptyBox, math.Inf(-1)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.input.Height()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestBox_midX(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		b := EmptyBox

		actual := b.midX()

		assert.True(t, math.IsNaN(actual))
	})

	testCases := []struct {
		name     string
		input    Box
		expected float64
	}{
		{"Zero", Box{}, 0},
		{"Negative", Box{-1, -2, 0, 0}, -0.5},
		{"Positive", Box{0, 0, 1, 2}, 0.5},
		{"Straddling", Box{-2, -1, 2, 1}, 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.input.midX()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestBox_midY(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		b := EmptyBox

		actual := b.midY()

		assert.True(t, math.IsNaN(actual))
	})

	testCases := []struct {
		name     string
		input    Box
		expected float64
	}{
		{"Zero", Box{}, 0},
		{"Negative", Box{-1, -2, 0, 0}, -1},
		{"Positive", Box{0, 0, 1, 2}, 1},
		{"Straddling", Box{-2, -1, 2, 1}, 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.input.midY()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestBox_Expand(t *testing.T) {
	testCases := []struct {
		name           string
		b, c, expected Box
	}{
		{"Zero", Box{}, Box{}, Box{}},
		{"Empty", EmptyBox, EmptyBox, EmptyBox},
		{"ZeroByEmpty", Box{}, EmptyBox, Box{}},
		{"EmptyByZero", EmptyBox, Box{}, Box{}},
		{"EmptyByUnit", EmptyBox, Box{-1, -1, 1, 1}, Box{-1, -1, 1, 1}},
		{"GrowXMin", Box{-1, -1, 1, 1}, Box{-2, -0.5, 0, 0.5}, Box{-2, -1, 1, 1}},
		{"GrowYMin", Box{-1, -1, 1, 1}, Box{-0.5, -2, 0, 0.5}, Box{-1, -2, 1, 1}},
		{"GrowXMax", Box{-1, -1, 1, 1}, Box{-0.5, -0.5, 2, 0.5}, Box{-1, -1, 2, 1}},
		{"GrowYMax", Box{-1, -1, 1, 1}, Box{-0.5, -0.5, 0.5, 2}, Box{-1, -1, 1, 2}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			b, c := testCase.b, testCase.c

			b.Expand(&c)

			assert.Equal(t, testCase.c, c, "Parameter box must not change.")
			assert.Equal(t, testCase.expected, b)
		})
	}
}

func TestBox_ExpandXY(t *testing.T) {
	testCases := []struct {
		name     string
		b        Box
		x, y     float64
		expected Box
	}{
		{"Zero", Box{}, 0, 0, Box{}},
		{"Empty", EmptyBox, 0, 0, Box{}},
		{"Unchanged", Box{0, 0, 1, 1}, 0.5, 0.5, Box{0, 0, 1, 1}},
		{"Left", Box{-1, -1, 1, 1}, -2, 0, Box{-2, -1, 1, 1}},
		{"Down", Box{-1, -1, 1, 1}, 0, -2, Box{-1, -2, 1, 1}},
		{"Right", Box{-1, -1, 1, 1}, 2, 0, Box{-1, -1, 2, 1}},
		{"Up", Box{-1, -1, 1, 1}, 0, 2, Box{-1, -1, 1, 2}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			b := testCase.b

			b.ExpandXY(testCase.x, testCase.y)

			assert.Equal(t, testCase.expected, b)
		})
	}
}

func TestBox_intersects(t *testing.T) {
	testCases := []struct {
		name     string
		b, c     Box
		expected bool
	}{
		{"Zero", Box{}, Box{}, true},
		{"Empty", EmptyBox, EmptyBox, false},
		{"ZeroEmpty", Box{}, EmptyBox, false},
		{"EmptyZero", EmptyBox, Box{}, false},
		{"FullyContained", Box{-2, -2, 2, 2}, Box{-1, -1, 1, 1}, true},
		{"OverlapLeft", Box{-2, -2, 2, 2}, Box{-3, -1, -2, 1}, true},
		{"OverlapDown", Box{-2, -2, 2, 2}, Box{1, -3, -1, -2}, true},
		{"OverlapRight", Box{-2, -2, 2, 2}, Box{2, -1, 3, 1}, true},
		{"OverlapUp", Box{-2, -2, 2, 2}, Box{1, 2, -1, 3}, true},
		{"IsLeftOf", Box{-2, -2, 0, 0}, Box{-100, -2, -50, 0}, false},
		{"IsBelow", Box{-2, -2, 0, 0}, Box{-2, -100, 0, -50}, false},
		{"IsRightOf", Box{-2, -2, 0, 2}, Box{50, -2, 100, 1}, false},
		{"IsAbove", Box{-2, -2, 2, 2}, Box{1, 50, 2, 100}, false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			b, c := testCase.b, testCase.c

			actual := b.intersects(&c)

			assert.Equal(t, testCase.expected, actual)
		})
	}
}
