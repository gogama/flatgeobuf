package packedrtree

import (
	"container/heap"
	"io"
	"math"
	"unsafe"
)

// A Ref is a single item within the PackedRTree and represents a
// reference to a feature stored in the data section. Each Ref consists
// of its feature's Offset into the data section plus a Box representing
// the bounding box of the feature's geometry.
type Ref struct {
	Box

	// Offset is the referenced feature's byte offset into the data
	// section.
	Offset int64
}

// A node is a private version of Ref used to (hopefully) reduce
// confusion. A leaf node is exactly the same as a Ref and has the
// same meaning. A non-leaf node is subtly different: the Box is the
// extent of the entire subtree rooted at the non-leaf node; and the
// Offset represents the node index of the node's first child node.
type node struct {
	Ref
}

const numNodeBytes = unsafe.Sizeof(node{})

func validateParams(numRefs int, nodeSize uint16) {
	if numRefs < 1 {
		textPanic("empty tree not allowed (num refs must be > 0)")
	} else if nodeSize < 2 {
		textPanic("node size must be at least 2")
	}
}

// Size returns the disk size in bytes of a packed Hilbert R-Tree index
// having a given feature reference count and node size. Panics if
// numRefs is less than 1 or nodeSize is less than 2, and returns an
// error if integer overflow occurs.
func Size(numRefs int, nodeSize uint16) (int64, error) {
	validateParams(numRefs, nodeSize)
	return size(numRefs, int(nodeSize))
}

// size returns the disk size in bytes of a packed Hilbert R-Tree index
// having a given feature reference count and node size. Returns an
// error if integer overflow occurs.
func size(numRefs, nodeSize int) (int64, error) {
	// Count total number of internal nodes in the tree.
	var numInternal int
	nodesThisLevel := numRefs
	for {
		nodesThisLevel = (nodesThisLevel + nodeSize - 1) / nodeSize
		numInternal += nodesThisLevel
		if nodesThisLevel == 1 {
			break
		}
	}

	// Calculate total number of nodes, ensuring it does not overflow
	// int.
	numNodes, err := totalNodes(numRefs, numInternal)
	if err != nil {
		return 0, err
	}

	// Ensure total tree size does not overflow int64.
	if int64(numNodes) > math.MaxInt64/int64(numNodeBytes) {
		return 0, textErr("index size overflows int64")
	}

	// Calculate and return total tree size.
	return int64(numNodes) * int64(numNodeBytes), nil
}

// totalNodes sums numRefs and numInternal, returning an error if
// integer overflow occurs.
func totalNodes(numRefs, numInternal int) (n int, err error) {
	n = numRefs + numInternal
	if n < numRefs || n < numInternal {
		n = 0
		err = textErr("total node count overflows int")
	}
	return
}

// A levelRange represents the range of node indices that comprise a
// level. Each levelRange is a closed/open node index pair [start, end)
// where start is the index (into packedRTree's nodes list) of the first
// node in the level and end is the index that is one past the last node
// in the level.
type levelRange struct {
	start, end int
}

// levelify creates the list of levelRange structures which
// deterministically results from a given leaf node count (numRefs) and
// child node count (nodeSize).
//
// In the official FlatGeobuf implementations, levelify is most
// analogous to the function or method named generateLevelBounds().
//
// For example, assume numRefs = 4, nodeSize = 2. The output of this
// function will be [[3, 7], [1, 3], [0, 1]], where first item in the
// list represents the leaf node level, and the last item in the list is
// the root level.
func levelify(numRefs, nodeSize int) ([]levelRange, error) {
	// numInternal is the number of internal nodes in the tree, a number
	// strictly less than numRefs.
	var numInternal int

	// Generate a list of node counts per level, in the same order as
	// the final levelRange list, i.e. the leaf level 0 is first and the
	// root level is last.
	//
	// Keeping with the example numRefs = 4, nodeSize = 2, the result of
	// this logic is nodesPerLevel = [4, 2, 1].
	nodesThisLevel := numRefs
	nodesPerLevel := make([]int, 1, 16)
	nodesPerLevel[0] = nodesThisLevel
	for {
		nodesThisLevel = (nodesThisLevel + nodeSize - 1) / nodeSize
		nodesPerLevel = append(nodesPerLevel, nodesThisLevel)
		numInternal += nodesThisLevel
		if nodesThisLevel == 1 {
			break
		}
	}

	// Sum up the total number of nodes.
	numNodes, err := totalNodes(numRefs, numInternal)
	if err != nil {
		return nil, err
	}

	// Generate a list of node start indices per level, in the same
	// order as the final levelRange list.
	//
	// Keeping with the example numRefs = 4, nodeSize = 2, the result of
	// this logic is levelIndices = [3, 1, 0].
	levelIndices := make([]int, len(nodesPerLevel))
	nodesRemaining := numNodes
	for i := range nodesPerLevel {
		nodesRemaining -= nodesPerLevel[i]
		levelIndices[i] = nodesRemaining
	}

	// Generate and return the final list of levelRange structures.
	levels := make([]levelRange, len(levelIndices))
	for i := range levelIndices {
		levels[i].start = levelIndices[i]
		levels[i].end = levelIndices[i] + nodesPerLevel[i]
	}
	return levels, nil
}

// A fetchFunc is used to fetch the nodes from the closed/open index
// range [i, j) into the target node array. It is used by packedRTree
// for streaming index searches.
type fetchFunc func(i, j int, nodes []node) error

// A ticket is a pending work item to be executed during a packedRTree
// search loop.
type ticket struct {
	// nodeIndex is the index of the first node to search.
	nodeIndex int
	// level is the R-Tree level that nodeIndex belongs to. Recall that
	// level 0 contains the leaf nodes.
	level int
}

// A ticketBag is a collection of pending work items to be executed
// during a packedRTree search loop.
//
// The reason type is a "bag" and not, for example, a queue, is that it
// can have arbitrary behavior defined by the packedRTree's pushFunc and
// popFunc. When performing a streaming search, the Seek function wants
// to traverse the index in sequential order, so ticketBag behaves like
// a min-heap (and implements heap.Interface for this purpose). When
// performing a search of static data contained in a PackedRTree,
// ticketBag behaves like a stack.
type ticketBag []ticket

func (tq ticketBag) Len() int            { return len(tq) }
func (tq ticketBag) Less(i, j int) bool  { return tq[i].nodeIndex < tq[j].nodeIndex }
func (tq ticketBag) Swap(i, j int)       { tq[i], tq[j] = tq[j], tq[i] }
func (tq *ticketBag) Push(x interface{}) { *tq = append(*tq, x.(ticket)) }
func (tq *ticketBag) Pop() interface{} {
	return stackPop(tq)
}

type pushFunc func(tq *ticketBag, t ticket)
type popFunc func(tq *ticketBag) ticket

func stackPush(tq *ticketBag, t ticket) {
	*tq = append(*tq, t)
}

func stackPop(tq *ticketBag) ticket {
	old := *tq
	n := len(old)
	x := old[n-1]
	*tq = old[0 : n-1]
	return x
}

func heapPush(tq *ticketBag, t ticket) {
	heap.Push(tq, t)
}

func heapPop(tq *ticketBag) ticket {
	return heap.Pop(tq).(ticket)
}

// A packedRTree is a private type which carries most of the generic
// functionality required by PackedRTree and Seek. Unlike PackedRTree, a
// packedRTree is capable of streaming index search.
type packedRTree struct {
	// numRefs is the number of leaf nodes, i.e. Ref values, in the
	// tree.
	numRefs int
	// nodeSize is the number of child nodes per parent node.
	nodeSize int
	// levels is the list of levelRange boundaries. Note that in keeping with
	// the other Flatgeofbuf implementations (and the Hilbert R-Tree at
	// https://en.wikipedia.org/wiki/Hilbert_R-tree), the leaf nodes are
	// at levelRange 0 and the root node is at len(levels)-1.
	levels []levelRange
	// nodes is the complete list of nodes in the tree, including
	// internal and leaf nodes.
	nodes []node
	// push is the function used to push a work ticket into a ticketBag
	// when executing a tree search. It may not be nil.
	push pushFunc
	// pop is the function used to pop the next work ticket from a
	// ticketBag when executing a tree search. It may not be nil.
	pop popFunc
	// fetch is the function used to fetch missing nodes into the nodes
	// slice for streaming index search use cases. If all nodes are
	// present from the beginning, fetch is nil.
	fetch fetchFunc
}

// noo constructs a new packedRTree.
//
// In the official FlatGeobuf implementations, noo is most analogous to
// the function or method named init().
func noo(numRefs int, nodeSize uint16, push pushFunc, pop popFunc, fetch fetchFunc) (packedRTree, error) {
	validateParams(numRefs, nodeSize)

	levels, err := levelify(numRefs, int(nodeSize))
	if err != nil {
		return packedRTree{}, err
	}

	return packedRTree{
		numRefs:  numRefs,
		nodeSize: int(nodeSize),
		levels:   levels,
		nodes:    make([]node, levels[0].end),
		push:     push,
		pop:      pop,
		fetch:    fetch,
	}, nil
}

// Result is a single index search result. A Result's fields can be used
// to locate the corresponding feature in the main data section of the
// FlatGeobuf file, or in the Ref list passed to New when creating the
// PackedRTree.
type Result struct {
	// Offset is the result feature's byte offset into the data section.
	Offset int64
	// RefIndex of the feature reference in the Hilbert-sorted list of
	// Ref values passed to New when creating the PackedRTree.
	RefIndex int
}

// search implements a generic Hilbert R-Tree search function which is
// capable of streaming search depending on the callback functions
// configured in prt.
func (prt *packedRTree) search(b Box) ([]Result, error) {
	q := make(ticketBag, 1)
	q[0] = ticket{nodeIndex: 0, level: len(prt.levels) - 1}
	r := make([]Result, 0)

	for {
		// Pop the next work ticket from the front of queue.
		t := prt.pop(&q)
		// Find the end node index to search this iteration and decide
		// if the target nodes to search are leaves.
		end := t.nodeIndex + prt.nodeSize
		if prt.levels[t.level].end < end {
			end = prt.levels[t.level].end
		}
		isLeafLevel := t.nodeIndex >= prt.levels[0].start
		// Fetch the nodes to be searched if they aren't yet available.
		if prt.fetch != nil {
			err := prt.fetch(t.nodeIndex, end, prt.nodes)
			if err != nil {
				return nil, err
			}
		}
		// Search the nodes.
		for pos := t.nodeIndex; pos < end; pos++ {
			n := &prt.nodes[pos]
			if !b.intersects(&n.Box) {
				continue
			} else if isLeafLevel {
				r = append(r, Result{Offset: n.Offset, RefIndex: pos - prt.levels[0].start})
			} else {
				prt.push(&q, ticket{nodeIndex: int(n.Offset), level: t.level - 1})
			}
		}
		// Stop and return if there is no remaining work.
		if len(q) == 0 {
			return r, nil
		}
	}
}

// PackedRTree is a packed Hilbert R-Tree.
type PackedRTree struct {
	packedRTree
}

// New creates a new packed Hilbert R-Tree from a non-empty,
// Hilbert-sorted list of feature references and a given R-Tree node
// size. Panics if the reference list is empty or node size is less than
// 2.
//
// Use HilbertSort to sort the feature references. If the input slice is
// not Hilbert-sorted, the behavior of the new PackedRTree is undefined.
func New(refs []Ref, nodeSize uint16) (*PackedRTree, error) {
	// Create the private, non-exported data structure.
	prt, err := noo(len(refs), nodeSize, stackPush, stackPop, nil)
	if err != nil {
		return nil, err
	}
	// Save copies of the leaf nodes.
	i := prt.levels[0].start
	for j := range refs {
		prt.nodes[i] = node{refs[j]}
		i++
	}
	// Generate the internal nodes.
	for i = 0; i < len(prt.levels)-1; i++ {
		level := prt.levels[i]
		nodeIndex := level.start
		parent := &prt.nodes[prt.levels[i+1].start]
		for nodeIndex < level.end {
			*parent = node{Ref: Ref{Null, int64(nodeIndex)}}
			var j int
			for {
				parent.expand(&prt.nodes[nodeIndex].Box)
				j++
				nodeIndex++
				if j == prt.nodeSize || nodeIndex == level.end {
					break
				}
			}
		}
	}
	// Return the exported data structure.
	return &PackedRTree{prt}, nil
}

// Bounds returns the bounding box around all features referenced by the
// packed Hilbert R-Tree.
func (prt *PackedRTree) Bounds() Box {
	return prt.nodes[len(prt.nodes)-1].Box
}

// Search searches the packed Hilbert R-Tree for qualified matches
// whose bounding rectangles intersect the query box.
//
// To directly search the index section of FlatGeobuf file without
// creating a PackedRTree, consider using the Seek function.
func (prt *PackedRTree) Search(b Box) []Result {
	r, err := prt.search(b)
	if err != nil {
		panic(err) // prt.search should never return error in this case.
	}
	return r
}

// Marshal serializes the packed Hilbert R-Tree as a FlatGeobuf index
// section.
//
// If you are writing a complete FlatGeobuf file, the writer should be
// positioned ready to write the first byte of the index. If this method
// returns without error, the writer will be positioned ready to write
// the first byte of the data section.
func (prt *PackedRTree) Marshal(w io.Writer) error {
	// TODO: For now can use encoding/binary's BigEndian to write the byte.
	// TODO: I would be inclined to use one of the available tricks to
	//       determine the endianness of the platform. If it already
	//       BigEndian just do the closest thing to a byte dump, otherwise
	//       use encoding/binary. Except encoding binary may be ultraslow...
	//       Maybe there's something else?
	//
	// TODO: Another option is to use https://github.com/yalue/native_endian
	//       but I would be more inclined to borrow its approach (find
	//       out how it used compile-time build tags to solve endianness).
	//
}

func Unmarshal(r io.Reader) (*PackedRTree, error) {
	// TODO: This will be the opposite of Marshal, in that it will
	//       read the whole thing including the internal nodes.
}

// TODO: Docs
func Seek(rs io.ReadSeeker, numRefs int, nodeSize uint16, b Box) ([]Result, error) {
	fetch := func(i, j int, nodes []node) error {
		// TODO: Seek to the position.

		// TODO: Read from the read seeker into nodes

		// TODO: Rust version has a nifty thing where it seeks past the
		//       remainder of the index at the end. Worth it? I think
		//       so because that's how Seek can be used from Reader
		//       without messing anything up.
		//           https://github.com/flatgeobuf/flatgeobuf/blob/master/src/rust/src/packed_r_tree.rs#L532-L535
		return nil
	}
	// Construct the private data structure using a min-heap for the
	// work tracking ticket bag to ensure the index is read
	// sequentially.
	prt, err := noo(numRefs, nodeSize, heapPush, heapPop, fetch)
	if err != nil {
		return nil, err
	}

	// TODO: For this function to work with flatgeobuf.Reader, we should
	//       ensure that after the index search is finished it seeks to
	//       the end of the index section, i.e. so that calls to r.Rewind()
	//       and r.DataSearch() work properly.
	return prt.search(b)
}
