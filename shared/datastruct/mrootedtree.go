package datastruct

import (
	"errors"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
)

type Element interface {
	Encode() []byte
	New(r io.Reader) Element

	// Returns this node unique id
	Id() string
}

type EmptyElement struct {
}

func (EmptyElement) Id() string {
	return strconv.Itoa(rand.Int())
}

func (EmptyElement) Encode() []byte {
	return []byte{1}
}

func (EmptyElement) New(r io.Reader) Element {
	return EmptyElement{}
}

type Node struct {
	NodeId string
	Height uint64
	child *Node
	Value Element
	Parents []*Node
}

func (n *Node) Next() (*Node)  {
	return n.child
}

var lg = log.New(os.Stdout, "mRootTree: ", log.Lshortfile|log.Lmicroseconds)


type MRootTree struct {
	// The height of the tree
	Height uint64

	// all of the roots
	roots []*Node
	// fast way to access roots
	rootsFasS map[*Node]int

	// node id -> to multiple nodes (sometimes they collide if we are looking for hashes)
	nodes map[string]*Node

	// Head of the longest chain, get
	// through GetLongestChain
	longestChainHead *Node
}

func (t *MRootTree) Find(id string) (*Node, bool){
	v, ok := t.nodes[id]
	return v, ok
}


// adds an element to the tree given a root, if the head is not a root
// then, we will add a new root to the tree, head can be nil
func (t *MRootTree) PrependElement(e Element, head *Node) (*Node, error) {
	var newNode Node

	if head != nil {
		newNode = Node{
			Value: e,
			child: head,
			Height: head.Height + 1,
			NodeId: e.Id(),
			Parents: make([]*Node, 0, 1),
		}
		head.Parents = append(head.Parents, &newNode)
	} else {
		newNode = Node{
			Value: e,
			child: nil,
			Height: 0,
			NodeId: e.Id(),
			Parents: make([]*Node, 0, 1),
		}
	}

	// the node id is the same as the node hash which sometimes collides so we want to handle that case as well
	if _, ok := t.nodes[newNode.NodeId]; ok {
		return nil, errors.New("cannot add node to tree as there is another node with the same hash")
	} else {
		t.nodes[newNode.NodeId] = &newNode
	}


	// append to map and root keeper
	if idx, ok := t.rootsFasS[head]; ok {
		t.roots[idx] = &newNode
		delete(t.rootsFasS, head)
		t.rootsFasS[&newNode] = idx
	} else {
		lg.Printf("Adding new root: %v", e.Id())
		t.roots = append(t.roots, &newNode)
		t.rootsFasS[&newNode] = len(t.roots) - 1
	}

	// check who is the longest
	if newNode.Height > t.Height {
		lg.Printf("New height %v", newNode.Height)
		t.longestChainHead = &newNode
		t.Height = newNode.Height
	}

	return &newNode, nil
}

// Gets all of the roots of the tree
func (t *MRootTree) GetRoots() ([]*Node) {
	return t.roots[:]
}

// Gets the longest chain, if the length of two chains is exactly the same
// then we return either one of them
func (t *MRootTree) GetLongestChain() *Node {
	return t.longestChainHead
}

func NewMRootTree() *MRootTree {
	v := new(MRootTree)
	v.roots = make([]*Node, 0, 10)
	v.rootsFasS = make(map[*Node]int)
	v.Height = 0
	v.nodes = make(map[string]*Node)

	return v
}