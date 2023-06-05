package packedrtree

import "io"

type PackedRTree struct {
	// extent is the minimum bounding rectangle around all items in the
	// tree.
	extent Box
	// nodes is the complete list of nodes in the tree, including both
	// leaf and non-leaf nodes.
	nodes []Node
	// levels is a list of the sub-lists of nodes which comprise each
	// level in the tree, where level zero represents the root of the
	// tree. Each sub-list in levels is a re-slicing of the relevant
	// part of nodes. The last sub-list comprises the leaf nodes.
	levels [][]Node
}

/**

numItems is number of leaf nodes.
numNodes is total number of nodes including internal and leaf nodes.
numNodes-numItems is index of the first leaf node.

levelNumNodes is number of nodes per level in reverse order (leaves first)
    = [numItems, numItems/nodeSize, numItems/nodeSize^2, ..., 1]
levelOffsets is XXXXX in forward order (root first).
    = [numNodes - numItems, numNodes - numItems - numItems/nodeSize, ... ]
    = [number of internal nodes all levels, number of internal nodes, all but last level, ...numItems]
levelBounds is YYYYYY in reverse order again so that the first entry is the set of leaves, and the last entry is the root node.
    = [
		 [numNodes - numItems, numNodes],                                 // length = numItems
		 [numNodes - numItems - numItems/nodeSize, numNodes - numItems],  // length = numItems/nodeSize
         [0, nodeSize]
      ]
*/

func New(items []Node, nodeSize uint16) (*PackedRTree, error) {
	// Validate inputs.
	if nodeSize < 2 {
		// TODO: Error out with "node size must be at least 2"
	}
	// TODO: All Flatgeobuf reference implementations return an error
	//       if len(items) < 1. Is that necessary, or is it possible to
	//       implement the construction and search gracefully such that
	//       zero is just one of several cases?

	// TODO: The way init() uses HILBERT_MAX as a parameter in calculating
	//       the final nodeSize allows nodeSize to be an odd number. IS
	//       that desirable? Don't know!

	var prt PackedRTree

	// Calculate the levels TODO better comment.
	prt.generateLevels()

	// Copy the given items into the relevant TODO.
	leaves := prt.levels[len(prt.levels)-1]
	for i := range leaves {
		leaves[i] = items[i]
	}
}

func (prt *PackedRTree) Search(bounds Box) []Result {
	// TODO: It's very instructive to look at the Rust implementation
	//       of search_stream because it only reads nodes within the
	//       actual search loop. So the way it works can tell us a lot
	//       about the level bounds etc.
}

// TODO: Documentation note: I didn't call this SearchStream because I
//
//	would argue that a seekable entity isn't really a stream.
func Search(r io.ReadSeeker, n int, nodeSize uint16, bounds Box) ([]Result, error) {
	// TODO: I would ideally like to build a PRT out of this and just
	//       call its Search method, if the same can be achieved without
	//       changing the time complexity. However, Rust implementation
	//       suggests this *might* result in over-reading?
}
