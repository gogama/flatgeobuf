package packedrtree

import (
	"container/heap"
	"io"
)

// A Ref is a single item within the PackedRTree and represents a
// reference to a feature stored in the data section. Each Ref consists
// of its feature's Offset into the data section plus a Box representing
// the bounding box of the feature's geometry.
type Ref struct {
	Box
	Offset int
}

// A node is an internal version of Ref used to (hopefully) reduce
// confusion. A leaf node is exactly the same as a Ref and has the
// same meaning. For an internal node, the Box is the bounding box of
// the entire internal node; and the Offset represents the node index
// of the node's first child node.
type node struct {
	Ref
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
// In the official Flatgeobuf implementations, levelify is most
// analogous to the function or method named generateLevelBounds().
//
// For example, assume numRefs = 4, nodeSize = 2. The output of this
// function will be [[3, 7], [1, 3], [0, 1]], where first item in the
// list represents the leaf node level, and the last item in the list is
// the root level.
func levelify(numRefs, nodeSize int) []levelRange {
	// numNodes is total count of internal and leaf nodes in the tree.
	// We initialize it with the number of leaf nodes.
	numNodes := numRefs

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
		numNodes += nodesThisLevel
		if nodesThisLevel == 1 {
			break
		}
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
	return levels
}

// A fetchFunc is used to fetch the nodes from the closed/open index
// range [i, j) into the target node array. It is used by packedRTree
// for streaming index searches.
type fetchFunc func(i, j int, nodes []node) error

// TODO: docs
type ticket struct {
	nodeIndex int
	level     int
}

// TODO: docs
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

// A packedRTree is an internal type which carries most of the generic
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
	// https://en.wikipedia.org/wiki/Hilbert_R-tree, the leaf nodes are
	// at levelRange 0 and the root node is at len(levels)-1.
	levels []levelRange
	// nodes is the complete list of nodes in the tree, including
	// internal and leaf nodes.
	nodes []node
	// TODO
	push pushFunc
	// TODO
	pop popFunc
	// fetch is the function used to fetch missing nodes into the nodes
	// slice for streaming index search use cases. If all nodes are
	// present from the beginning, fetch is nil.
	fetch fetchFunc
}

// new constructs a new packedRTree.
//
// In the official Flatgeobuf implementations, new is most analogous to
// the function or method named init().
func new(numRefs int, nodeSize uint16, push pushFunc, pop popFunc, fetch fetchFunc) (packedRTree, error) {

	var err error // TODO: Error handling.
	if err != nil {
		return packedRTree{}, err
	}

	levels := levelify(numRefs, int(nodeSize))

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
				r = append(r, Result{Offset: n.Offset, Index: pos - prt.levels[0].i})
			} else {
				prt.push(&q, ticket{nodeIndex: n.Offset, level: t.level - 1})
			}
		}
		// Stop and return if there is no remaining work.
		if len(q) == 0 {
			return r, nil
		}
	}
}

// TODO: Docs
type PackedRTree struct {
	packedRTree
}

// TODO: Docs
func New(refs []Ref, nodeSize uint16) (*PackedRTree, error) {
	// Create the internal data structure.
	prt, err := new(len(refs), nodeSize, stackPush, stackPop, nil)
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
			*parent = node{Ref: Ref{Null, nodeIndex}}
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
	// Return external data structure.
	return &PackedRTree{prt}, nil
}

// TODO: Docs
func (prt *PackedRTree) Bounds() Box {
	return prt.nodes[len(prt.nodes)-1].Box
}

// TODO: docs
func (prt *PackedRTree) Search(b Box) []Result {
	r, err := prt.search(b)
	if err != nil {
		// TODO: Panic here, as should never have an error.
	}
	return r
}

func (prt *PackedRTree) Marshal(w io.Writer) error {
	// TODO
}

func Unmarshal(r io.Reader) (*PackedRTree, error) {
	// TODO: This will be t he opposite of WriteTo, in that it will
	//       read the whole thing including the internal nodes.
}

// TODO: Docs
func Seek(r io.ReadSeeker, numRefs int, nodeSize uint16, b Box) ([]Result, error) {
	fetch := func(i, j int, nodes []node) error {
		// TODO: Read from the read seeker into nodes

		// TODO: Rust version has a nifty thing where it seeks past the
		//       remainder of the index at the end. Worth it?
		//           https://github.com/flatgeobuf/flatgeobuf/blob/master/src/rust/src/packed_r_tree.rs#L532-L535
		return nil
	}
	// Construct the internal data structure using a min-heap for the
	// work tracking ticket bag to ensure the index is read
	// sequentially.
	prt, err := new(numRefs, nodeSize, heapPush, heapPop, fetch)
	if err != nil {
		return nil, err
	}
	return prt.search(b)
}
