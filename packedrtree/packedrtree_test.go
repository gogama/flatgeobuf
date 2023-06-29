// Copyright 2023 The flatgeobuf (Go) Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package packedrtree

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRef_String(t *testing.T) {
	testCases := []struct {
		name     string
		input    Ref
		expected string
	}{
		{"Zero", Ref{}, "Ref{[0,0,0,0],Offset:0}"},
		{"Integers", Ref{Box: Box{-1, 2, -3, 4}, Offset: -5}, "Ref{[-1,2,-3,4],Offset:-5}"},
		{"Exact", Ref{Box: Box{-100.5, -200.25, 1234.125, 5678.0625}, Offset: 6111}, "Ref{[-100.5,-200.25,1234.125,5678.0625],Offset:6111}"},
		{"Rounded", Ref{Box: Box{-100000.0625, 123.015625, 99.0078125, -2.001953125}, Offset: -12345678}, "Ref{[-100000.06,123.01562,99.007812,-2.0019531],Offset:-12345678}"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := testCase.input.String()

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestSize(t *testing.T) {
	t.Run("Panic", func(t *testing.T) {
		testCases := []struct {
			name     string
			numRefs  int
			nodeSize uint16
			expected string
		}{
			{
				name:     "numRefs.Zero",
				numRefs:  0,
				nodeSize: 2,
				expected: "packedrtree: empty tree not allowed (num refs must be > 0)",
			},
			{
				name:     "numRefs.Negative",
				numRefs:  -1,
				nodeSize: 2,
				expected: "packedrtree: empty tree not allowed (num refs must be > 0)",
			},
			{
				name:     "nodeSize.Zero",
				numRefs:  1,
				nodeSize: 0,
				expected: "packedrtree: node size must be at least 2",
			},
			{
				name:     "nodeSize.One",
				numRefs:  1,
				nodeSize: 1,
				expected: "packedrtree: node size must be at least 2",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.PanicsWithValue(t, testCase.expected, func() {
					_, _ = Size(testCase.numRefs, testCase.nodeSize)
				})
			})
		}
	})

	t.Run("Error", func(t *testing.T) {
		testCases := []struct {
			name            string
			numRefs         int
			nodeSize        uint16
			expected        string
			require64BitInt bool
		}{
			{
				name:     "NodeCountOverflowsInt",
				numRefs:  math.MaxInt,
				nodeSize: 2,
				expected: "packedrtree: total node count overflows int",
			},
			{
				name:            "IndexSizeOverflowsInt64",
				numRefs:         math.MaxInt / 32,
				nodeSize:        16,
				expected:        "packedrtree: index size overflows int",
				require64BitInt: true,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				if testCase.require64BitInt && math.MaxInt != math.MaxInt64 {
					t.Skip("Skipping: This test case requires 64 bit ints")
				}

				n, err := Size(testCase.numRefs, testCase.nodeSize)

				assert.Equal(t, 0, n)
				assert.EqualError(t, err, testCase.expected)
			})
		}
	})

	t.Run("Success", func(t *testing.T) {
		testCases := []struct {
			name     string
			numRefs  int
			nodeSize uint16
			expected int
		}{
			{
				name:     "Minimum",
				numRefs:  1,
				nodeSize: 2,
				expected: 2 * numNodeBytes,
			},
			{
				name:     "OneFullLevel",
				numRefs:  2,
				nodeSize: 2,
				expected: 3 * numNodeBytes,
			},
			{
				name:     "TwoFullLevels",
				numRefs:  4,
				nodeSize: 2,
				expected: 7 * numNodeBytes,
			},
			{
				name:     "ThreeFullLevels",
				numRefs:  8,
				nodeSize: 2,
				expected: 15 * numNodeBytes,
			},
			{
				name:     "Big",
				numRefs:  math.MaxInt32/32 - 1,
				nodeSize: 64,
				expected: (0x1 + 0x4 + 0x100 + 0x4000 + 0x100000 + (math.MaxInt32/32 - 1)) * numNodeBytes,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				n, err := Size(testCase.numRefs, testCase.nodeSize)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, n)
			})
		}
	})
}

func TestLevelify(t *testing.T) {
	t.Run("Panics", func(t *testing.T) {
		testCases := []struct {
			name     string
			numRefs  uint
			nodeSize uint
			expected string
		}{
			{
				name:     "NodeCountOverflowsInt",
				numRefs:  math.MaxInt,
				nodeSize: 2,
				expected: "packedrtree: total node count overflows int",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.PanicsWithError(t, testCase.expected, func() {
					_ = levelify(testCase.numRefs, testCase.nodeSize)
				})
			})
		}
	})

	t.Run("Success", func(t *testing.T) {
		testCases := []struct {
			name     string
			numRefs  uint
			nodeSize uint
			expected []levelRange
		}{
			{
				name:     "Minimum",
				numRefs:  1,
				nodeSize: 2,
				expected: []levelRange{{1, 2}, {0, 1}},
			},
			{
				name:     "OneFullLevel",
				numRefs:  2,
				nodeSize: 2,
				expected: []levelRange{{1, 3}, {0, 1}},
			},
			{
				name:     "TwoFullLevels",
				numRefs:  4,
				nodeSize: 2,
				expected: []levelRange{{3, 7}, {1, 3}, {0, 1}},
			},
			{
				name:     "ThreeFullLevels",
				numRefs:  8,
				nodeSize: 2,
				expected: []levelRange{{7, 15}, {3, 7}, {1, 3}, {0, 1}},
			},
			{
				name:     "Big",
				numRefs:  math.MaxInt32/32 - 1,
				nodeSize: 64,
				expected: []levelRange{{1065221, 68174083}, {16645, 1065221}, {261, 16645}, {5, 261}, {1, 5}, {0, 1}},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				levels := levelify(testCase.numRefs, testCase.nodeSize)

				assert.Equal(t, testCase.expected, levels)
				sz, err := size(testCase.numRefs, testCase.nodeSize)
				require.NoError(t, err)
				assert.Equal(t, testCase.expected[0].end, sz/numNodeBytes)
			})
		}
	})
}

func TestTicketBag(t *testing.T) {
	t.Run("FromZero", func(t *testing.T) {
		var q ticketBag
		expected := ticket{3, 5}

		t.Run("LenBefore", func(t *testing.T) {
			assert.Equal(t, 0, q.Len())
		})

		t.Run("Push", func(t *testing.T) {
			q.Push(expected)
		})

		t.Run("LenMiddle", func(t *testing.T) {
			assert.Equal(t, 1, q.Len())
		})

		t.Run("Pop", func(t *testing.T) {
			actual := q.Pop().(ticket)

			assert.Equal(t, expected, actual)
		})

		t.Run("LenAfter", func(t *testing.T) {
			assert.Equal(t, 0, q.Len())
		})
	})

	t.Run("FromMake", func(t *testing.T) {
		q := make(ticketBag, 1)
		n := 100

		t.Run("Before", func(t *testing.T) {
			assert.Equal(t, 1, q.Len())
		})

		t.Run("Push", func(t *testing.T) {
			for i := 1; i < n; i++ {
				q.Push(ticket{i, n*100 + i})

				assert.Equal(t, i+1, q.Len())
			}
		})

		t.Run("Reverse", func(t *testing.T) {
			for i := 0; i < n/2; i++ {
				assert.True(t, q.Less(i, n-i-1))

				q.Swap(i, n-i-1)

				assert.False(t, q.Less(i, n-i-1))
			}
		})

		t.Run("Pop", func(t *testing.T) {
			tk := q.Pop().(ticket)

			assert.Equal(t, ticket{}, tk)
			assert.Equal(t, n-1, q.Len())

			for i := 1; i < n; i++ {
				tk = q.Pop().(ticket)

				assert.Equal(t, ticket{i, n*100 + i}, tk)
				assert.Equal(t, n-i-1, q.Len())
			}
		})
	})
}

func TestHeap(t *testing.T) {
	var q ticketBag
	n := 8

	t.Run("Push", func(t *testing.T) {
		heapPush(&q, ticket{0, 0})
		heapPush(&q, ticket{3, 0})
		heapPush(&q, ticket{2, 0})
		heapPush(&q, ticket{5, 0})
		heapPush(&q, ticket{6, 0})
		heapPush(&q, ticket{1, 0})
		heapPush(&q, ticket{4, 0})
		heapPush(&q, ticket{7, 0})

		assert.Equal(t, n, q.Len())
	})

	t.Run("Pop", func(t *testing.T) {
		for i := 0; i < n; i++ {
			tk := heapPop(&q)

			assert.Equal(t, ticket{i, 0}, tk)
			assert.Equal(t, n-i-1, q.Len())
		}
	})
}

func TestResults(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		var rs Results

		assert.Equal(t, 0, rs.Len())
	})

	t.Run("Less", func(t *testing.T) {
		rs := Results{
			Result{0, 0},
			Result{1, 0},
		}

		assert.False(t, rs.Less(0, 0))
		assert.True(t, rs.Less(0, 1))
		assert.False(t, rs.Less(1, 0))
		assert.False(t, rs.Less(1, 1))
	})

	t.Run("Swap", func(t *testing.T) {
		rs1 := Results{
			Result{0, 0},
			Result{1, 0},
		}
		rs2 := make(Results, len(rs1))
		copy(rs2, rs1)

		rs1.Swap(0, 0)

		assert.Equal(t, rs2, rs1)

		rs1.Swap(1, 1)

		assert.Equal(t, rs2, rs1)

		rs1.Swap(0, 1)

		assert.Equal(t, Results{rs2[1], rs2[0]}, rs1)
	})

	t.Run("Sorts", func(t *testing.T) {
		m := 10
		for n := 0; n <= m; n++ {
			t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
				expected := make(Results, n)
				actual := make(Results, n)
				for i := 0; i < n; i++ {
					expected[i] = Result{int64(i), 0}
					actual[i] = Result{int64(i), 0}
				}
				r := rand.New(rand.NewSource(int64(n)))
				r.Shuffle(n, func(i, j int) {
					actual[i], actual[j] = actual[j], actual[i]
				})

				sort.Sort(actual)

				assert.Equal(t, expected, actual)
			})
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("Panics", func(t *testing.T) {
		testCases := []struct {
			name     string
			refs     []Ref
			nodeSize uint16
			expected string
		}{
			{
				name:     "numRefs.Nil",
				nodeSize: 2,
				expected: "packedrtree: empty tree not allowed (num refs must be > 0)",
			},
			{
				name:     "numRefs.Empty",
				refs:     make([]Ref, 0),
				nodeSize: 2,
				expected: "packedrtree: empty tree not allowed (num refs must be > 0)",
			},
			{
				name:     "nodeSize.Zero",
				refs:     make([]Ref, 1),
				nodeSize: 0,
				expected: "packedrtree: node size must be at least 2",
			},
			{
				name:     "nodeSize.One",
				refs:     make([]Ref, 1),
				nodeSize: 1,
				expected: "packedrtree: node size must be at least 2",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.PanicsWithValue(t, testCase.expected, func() {
					_, _ = New(testCase.refs, testCase.nodeSize)
				})
			})
		}
	})

	// We don't test the overflow error cases here because doing so
	// would require the test environment to do massive memory
	// allocations.

	t.Run("Success", func(t *testing.T) {
		// ...                   ^
		// ...                   |             [0]
		// ...                   |          [1]
		// ...                   |       [2]
		// ...                   |    [3]
		// ...                   | [4]
		// ...  <---------------[5]---------------->
		// ...               [6] |
		// ...            [7]    |
		// ...         [8]       |
		// ...      [9]          |
		// ...   [10]            v
		n := 11
		refs := make([]Ref, n)
		bounds := make([]Box, n)
		bounds[0] = EmptyBox
		for i := 0; i < n; i++ {
			if i > 0 {
				bounds[i] = bounds[i-1]
			}
			refs[i] = Ref{
				Box: Box{
					XMin: float64(n - 2*i - 2),
					YMin: float64(n - 2*i - 2),
					XMax: float64(n - 2*i),
					YMax: float64(n - 2*i),
				},
				Offset: int64(i),
			}
			bounds[i].Expand(&refs[i].Box)
		}

		t.Run("SneakyHilbertSortTest", func(t *testing.T) {
			expected := make([]Ref, n)
			copy(expected, refs)

			HilbertSort(refs, bounds[n-1])

			assert.Equal(t, expected, refs)
		})

		testCases := []struct {
			name     string
			numRefs  int
			nodeSize uint16
			nodes    []node
			levels   []levelRange
		}{
			{
				name:     "NodeSize2.Minimum",
				numRefs:  1,
				nodeSize: 2,
				levels:   []levelRange{{1, 2}, {0, 1}},
			},
			{
				name:     "NodeSize2.OneLevelFull",
				numRefs:  2,
				nodeSize: 2,
				levels:   []levelRange{{1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize2.TwoLevelsFull",
				numRefs:  4,
				nodeSize: 2,
				levels:   []levelRange{{3, 7}, {1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize2.ThreeLevelsFull",
				numRefs:  8,
				nodeSize: 2,
				levels:   []levelRange{{7, 15}, {3, 7}, {1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize3.Minimum",
				numRefs:  1,
				nodeSize: 3,
				levels:   []levelRange{{1, 2}, {0, 1}},
			},
			{
				name:     "NodeSize3.OneLevelPart",
				numRefs:  2,
				nodeSize: 3,
				levels:   []levelRange{{1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize3.OneLevelFull",
				numRefs:  3,
				nodeSize: 3,
				levels:   []levelRange{{1, 4}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.4Refs",
				numRefs:  4,
				nodeSize: 3,
				levels:   []levelRange{{3, 7}, {1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.5Refs",
				numRefs:  5,
				nodeSize: 3,
				levels:   []levelRange{{3, 8}, {1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.6Refs",
				numRefs:  6,
				nodeSize: 3,
				levels:   []levelRange{{3, 9}, {1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.7Refs",
				numRefs:  7,
				nodeSize: 3,
				levels:   []levelRange{{4, 11}, {1, 4}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.8Refs",
				numRefs:  8,
				nodeSize: 3,
				levels:   []levelRange{{4, 12}, {1, 4}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.9Refs",
				numRefs:  9,
				nodeSize: 3,
				levels:   []levelRange{{4, 13}, {1, 4}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.10Refs",
				numRefs:  10,
				nodeSize: 3,
				levels:   []levelRange{{7, 17}, {3, 7}, {1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize3.TwoLevels.11Refs",
				numRefs:  11,
				nodeSize: 3,
				levels:   []levelRange{{7, 18}, {3, 7}, {1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize4.Minimum",
				numRefs:  1,
				nodeSize: 4,
				levels:   []levelRange{{1, 2}, {0, 1}},
			},
			{
				name:     "NodeSize4.OneLevel.2Refs",
				numRefs:  2,
				nodeSize: 4,
				levels:   []levelRange{{1, 3}, {0, 1}},
			},
			{
				name:     "NodeSize5.OneLevel.11Refs",
				numRefs:  11,
				nodeSize: 5,
				levels:   []levelRange{{4, 15}, {1, 4}, {0, 1}},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				prt, err := New(refs[0:testCase.numRefs], testCase.nodeSize)

				require.NoError(t, err)
				require.NotNil(t, prt)

				t.Run("ValidateNew", func(t *testing.T) {
					assert.Equal(t, testCase.levels, prt.levels)
					assert.Equal(t, testCase.numRefs, prt.NumRefs())
					assert.Equal(t, bounds[testCase.numRefs-1], prt.Bounds())
				})

				t.Run("Search", func(t *testing.T) {
					t.Run("None", func(t *testing.T) {
						rs := prt.Search(EmptyBox)

						assert.Len(t, rs, 0)
					})

					t.Run("One", func(t *testing.T) {
						for i := 0; i < testCase.numRefs; i++ {
							t.Run(strconv.Itoa(i), func(t *testing.T) {
								b := Box{
									XMin: refs[i].Box.XMin + 0.00001,
									YMin: refs[i].Box.YMin + 0.00001,
									XMax: refs[i].Box.XMax - 0.00001,
									YMax: refs[i].Box.YMax - 0.00001,
								}

								rs := prt.Search(b)

								require.Len(t, rs, 1)
								assert.Equal(t, rs[0].Offset, int64(i))
							})
						}
					})

					t.Run("Some", func(t *testing.T) {
						for i := 0; i < testCase.numRefs; i++ {
							t.Run(strconv.Itoa(i), func(t *testing.T) {
								expected := make(Results, 0, 3)
								if i > 0 {
									expected = append(expected, Result{int64(i - 1), i - 1})
								}
								expected = append(expected, Result{int64(i), i})
								if i < testCase.numRefs-1 {
									expected = append(expected, Result{int64(i + 1), i + 1})
								}

								actual := prt.Search(refs[i].Box)

								assert.Len(t, actual, len(expected))
								sort.Sort(actual)
								assert.Equal(t, expected, actual)
							})
						}
					})

					t.Run("All", func(t *testing.T) {
						expected := make(Results, testCase.numRefs)
						for i := range expected {
							expected[i].RefIndex = i
							expected[i].Offset = int64(i)
						}

						actual := prt.Search(Box{
							XMin: math.Inf(-1),
							YMin: math.Inf(-1),
							XMax: math.Inf(1),
							YMax: math.Inf(1),
						})

						assert.Len(t, actual, testCase.numRefs)
						sort.Sort(actual)
						assert.Equal(t, expected, actual)
					})
				})

				var b bytes.Buffer

				t.Run("Marshal", func(t *testing.T) {
					var o int
					o, err = prt.Marshal(&b)
					require.NoError(t, err)

					var p int
					p, err = Size(testCase.numRefs, testCase.nodeSize)
					require.NoError(t, err)

					assert.Equal(t, p, o)
					assert.Equal(t, p, b.Len())
				})

				var qrt *PackedRTree

				t.Run("Unmarshal", func(t *testing.T) {
					qrt, err = Unmarshal(&b, testCase.numRefs, testCase.nodeSize)

					require.NoError(t, err)
					require.NotNil(t, qrt)
				})

				t.Run("ValidateUnmarshalled", func(t *testing.T) {
					assert.Equal(t, testCase.levels, qrt.levels)
					assert.Equal(t, testCase.numRefs, qrt.NumRefs())
					assert.Equal(t, bounds[testCase.numRefs-1], qrt.Bounds())
					assert.Equal(t, prt.nodes, qrt.nodes)
				})
			})
		}
	})
}

func TestUnmarshal(t *testing.T) {
	t.Run("Panic", func(t *testing.T) {
		testCases := []struct {
			name     string
			r        io.Reader
			numRefs  int
			nodeSize uint16
			expected string
		}{
			{
				name:     "r.nil",
				numRefs:  1,
				nodeSize: 2,
				expected: "packedrtree: nil reader",
			},
			{
				name:     "numRefs.Zero",
				r:        strings.NewReader("foo"),
				numRefs:  0,
				nodeSize: 2,
				expected: "packedrtree: empty tree not allowed (num refs must be > 0)",
			},
			{
				name:     "nodeSize.Zero",
				r:        strings.NewReader("bar"),
				numRefs:  1,
				nodeSize: 0,
				expected: "packedrtree: node size must be at least 2",
			},
			{
				name:     "nodeSize.One",
				r:        strings.NewReader("baz"),
				numRefs:  1,
				nodeSize: 1,
				expected: "packedrtree: node size must be at least 2",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.PanicsWithValue(t, testCase.expected, func() {
					_, _ = Unmarshal(testCase.r, testCase.numRefs, testCase.nodeSize)
				})
			})
		}
	})

	t.Run("Error", func(t *testing.T) {
		testCases := []struct {
			name            string
			setup           func(*mockReader)
			numRefs         int
			nodeSize        uint16
			expected        []interface{}
			require64BitInt bool
		}{
			{
				name:     "NodeCountOverflowsInt",
				numRefs:  math.MaxInt,
				nodeSize: 2,
				expected: []interface{}{"packedrtree: total node count overflows int"},
			},
			{
				name:            "IndexSizeOverflowsInt64",
				numRefs:         math.MaxInt / 32,
				nodeSize:        16,
				expected:        []interface{}{"packedrtree: index size overflows int"},
				require64BitInt: true,
			},
			{
				name: "UnexpectedEOF",
				setup: func(r *mockReader) {
					r.
						On("Read", mock.Anything).
						Return(0, io.ErrUnexpectedEOF).
						Once()
				},
				numRefs:  1,
				nodeSize: 2,
				expected: []interface{}{"packedrtree: failed to read index bytes: " + io.ErrUnexpectedEOF.Error(), io.ErrUnexpectedEOF},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				if testCase.require64BitInt && math.MaxInt != math.MaxInt64 {
					t.Skip("Skipping: This test case requires 64 bit ints")
				}

				var r mockReader
				r.Test(t)
				if testCase.setup != nil {
					testCase.setup(&r)
				}

				prt, err := Unmarshal(&r, testCase.numRefs, testCase.nodeSize)

				for i := range testCase.expected {
					switch x := testCase.expected[i].(type) {
					case string:
						assert.EqualError(t, err, x)
					case error:
						assert.ErrorIs(t, err, x)
					default:
						panic(x)
					}
				}
				assert.Nil(t, prt)
				r.AssertExpectations(t)
			})
		}
	})
}

func TestSeek(t *testing.T) {
	t.Run("Panic", func(t *testing.T) {
		testCases := []struct {
			name     string
			r        io.ReadSeeker
			numRefs  int
			nodeSize uint16
			expected string
		}{
			{
				name:     "r.nil",
				numRefs:  1,
				nodeSize: 2,
				expected: "packedrtree: nil read seeker",
			},
			{
				name:     "numRefs.Zero",
				r:        strings.NewReader("foo"),
				numRefs:  0,
				nodeSize: 2,
				expected: "packedrtree: empty tree not allowed (num refs must be > 0)",
			},
			{
				name:     "nodeSize.Zero",
				r:        strings.NewReader("bar"),
				numRefs:  1,
				nodeSize: 0,
				expected: "packedrtree: node size must be at least 2",
			},
			{
				name:     "nodeSize.One",
				r:        strings.NewReader("baz"),
				numRefs:  1,
				nodeSize: 1,
				expected: "packedrtree: node size must be at least 2",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.PanicsWithValue(t, testCase.expected, func() {
					_, _ = Seek(testCase.r, testCase.numRefs, testCase.nodeSize, Box{})
				})
			})
		}
	})

	t.Run("Error", func(t *testing.T) {
		// Create a byte array to read by marshalling an
		// index.
		// ...    ^
		// ...    |              [0]
		// ...    |
		// ...    |       [1]
		// ...    |
		// ...    | [2]
		// ...    +----------------->
		refs := []Ref{
			{Box: Box{XMin: 4, YMin: 4, XMax: 5, YMax: 5}, Offset: 1},
			{Box: Box{XMin: 2, YMin: 2, XMax: 3, YMax: 3}, Offset: 2},
			{Box: Box{XMin: 0, YMin: 0, XMax: 1, YMax: 1}, Offset: 3},
		}
		bounds := Box{0, 0, 5, 5}
		dup := make([]Ref, len(refs))
		copy(dup, refs)
		HilbertSort(refs, bounds)
		require.Equal(t, refs, dup)

		var err error
		var prt *PackedRTree
		prt, err = New(refs, 2)
		require.NoError(t, err)
		require.NotNil(t, prt)

		var buf bytes.Buffer
		_, err = prt.Marshal(&buf)
		require.NoError(t, err)

		b := buf.Bytes()

		// Run sub-test cases.
		testCases := []struct {
			name            string
			setup           func(*testing.T, *mockReader)
			numRefs         int
			nodeSize        uint16
			b               Box
			expected        []interface{}
			require64BitInt bool
		}{
			{
				name: "FailToCacheIndexStartOffset",
				setup: func(_ *testing.T, rs *mockReader) {
					rs.
						On("Seek", int64(0), io.SeekCurrent).
						Return(int64(0), io.ErrClosedPipe).
						Once()
				},
				numRefs:  2,
				nodeSize: 6,
				expected: []interface{}{"packedrtree: failed to cache index start offset: " + io.ErrClosedPipe.Error(), io.ErrClosedPipe},
			},
			{
				name: "NodeCountOverflowsInt",
				setup: func(_ *testing.T, rs *mockReader) {
					rs.
						On("Seek", int64(0), io.SeekCurrent).
						Return(int64(0), nil).
						Once()
				},
				numRefs:  math.MaxInt,
				nodeSize: 2,
				expected: []interface{}{"packedrtree: total node count overflows int"},
			},
			{
				name: "IndexSizeOverflowsInt64",
				setup: func(_ *testing.T, rs *mockReader) {
					rs.
						On("Seek", int64(0), io.SeekCurrent).
						Return(int64(0), nil).
						Once()
				},
				numRefs:         math.MaxInt / 32,
				nodeSize:        16,
				expected:        []interface{}{"packedrtree: index size overflows int"},
				require64BitInt: true,
			},
			{
				name: "IndexEndOverflowsInt64",
				setup: func(_ *testing.T, rs *mockReader) {
					rs.
						On("Seek", int64(0), io.SeekCurrent).
						Return(int64(math.MaxInt64), nil).
						Once()
				},
				numRefs:  2,
				nodeSize: 6,
				expected: []interface{}{"packedrtree: index end offset overflows int64"},
			},
			{
				name: "FailToReadInFetch",
				setup: func(_ *testing.T, rs *mockReader) {
					rs.
						On("Seek", int64(0), io.SeekCurrent).
						Return(int64(0), nil).
						Once()
					rs.
						On("Read", mock.Anything).
						Return(0, io.ErrUnexpectedEOF).
						Once()
				},
				numRefs:  2,
				nodeSize: 6,
				expected: []interface{}{"packedrtree: failed to read nodes [0..1), rel. offset 0: " + io.ErrUnexpectedEOF.Error(), io.ErrUnexpectedEOF},
			},
			{
				name: "FailToSeekInFetch",
				setup: func(t *testing.T, rs *mockReader) {
					// Cache the start offset.
					rs.
						On("Seek", int64(0), io.SeekCurrent).
						Return(int64(0), nil).
						Once()
					// Read root node.
					rs.
						On("Read", mock.MatchedBy(func(p []byte) bool { return len(p) == numNodeBytes })).
						Run(func(args mock.Arguments) {
							p := args.Get(0).([]byte)
							copy(p, b)
						}).
						Return(numNodeBytes, nil).
						Once()
					// Read second level.
					rs.
						On("Read", mock.MatchedBy(func(p []byte) bool { return len(p) == 2*numNodeBytes })).
						Run(func(args mock.Arguments) {
							p := args.Get(0).([]byte)
							copy(p, b[1*numNodeBytes:3*numNodeBytes])
						}).
						Return(2*numNodeBytes, nil).
						Once()
					//
					rs.
						On("Seek", int64(2*numNodeBytes), io.SeekCurrent).
						Return(int64(0), io.ErrUnexpectedEOF).
						Once()
				},
				numRefs:  prt.NumRefs(),
				nodeSize: prt.NodeSize(),
				b:        Box{XMin: 0.25, YMin: 0.25, XMax: 0.75, YMax: 0.75},
				expected: []interface{}{"packedrtree: failed to seek to node 5, rel. offset 80: " + io.ErrUnexpectedEOF.Error(), io.ErrUnexpectedEOF},
			},
			{
				name: "FailToSeekPastIndex",
				setup: func(t *testing.T, rs *mockReader) {
					// Cache the start offset.
					rs.
						On("Seek", int64(0), io.SeekCurrent).
						Return(int64(0), nil).
						Once()
					// Read root node.
					rs.
						On("Read", mock.MatchedBy(func(p []byte) bool { return len(p) == numNodeBytes })).
						Run(func(args mock.Arguments) {
							p := args.Get(0).([]byte)
							copy(p, b)
						}).
						Return(numNodeBytes, nil).
						Once()
					// Seek to end of index.
					rs.
						On("Seek", int64(6*numNodeBytes), io.SeekStart).
						Return(int64(0), io.ErrUnexpectedEOF).
						Once()
				},
				numRefs:  prt.NumRefs(),
				nodeSize: prt.NodeSize(),
				b:        EmptyBox,
				expected: []interface{}{"packedrtree: failed to skip to end of index after Seek: " + io.ErrUnexpectedEOF.Error(), io.ErrUnexpectedEOF},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				if testCase.require64BitInt && math.MaxInt != math.MaxInt64 {
					t.Skip("Skipping: This test case requires 64 bit ints")
				}

				var r mockReader
				r.Test(t)
				if testCase.setup != nil {
					testCase.setup(t, &r)
				}

				rs, err := Seek(&r, testCase.numRefs, testCase.nodeSize, testCase.b)

				for i := range testCase.expected {
					switch x := testCase.expected[i].(type) {
					case string:
						assert.EqualError(t, err, x)
					case error:
						assert.ErrorIs(t, err, x)
					default:
						panic(x)
					}
				}
				assert.Nil(t, rs)
				r.AssertExpectations(t)
			})
		}
	})
}

type mockReader struct {
	mock.Mock
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	args := r.Called(p)
	return args.Int(0), args.Error(1)
}

func (r *mockReader) Seek(offset int64, whence int) (int64, error) {
	args := r.Called(offset, whence)
	return args.Get(0).(int64), args.Error(1)
}
