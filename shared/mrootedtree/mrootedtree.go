package mrootedtree

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

type node struct {
	NodeId string
	Height uint64
	child *node
	Value Element
	Parents int
}

func (n *node) Next() (*node)  {
	return n.child
}

var lg = log.New(os.Stdout, "mRootTree: ", log.Lshortfile|log.Lmicroseconds)


type MRootTree struct {
	// The height of the tree
	Height uint64

	// all of the roots
	roots []*node
	// fast way to access roots
	rootsFasS map[*node]int

	// Head of the longest chain, get
	// through GetLongestChain
	longestChainHead *node
}

// adds an element to the tree given a root, if the head is not a root
// then, we will add a new root to the tree, head can be nil
func (t *MRootTree) PrependElement(e Element, head *node) (*node, error) {
	var newNode node

	if head != nil {
		newNode = node{
			Value: e,
			child: head,
			Height: head.Height + 1,
			NodeId: e.Id(),
			Parents: 0,
		}
		head.Parents += 1
	} else if len(t.roots) == 0 {
		newNode = node{
			Value: e,
			child: nil,
			Height: 0,
			NodeId: e.Id(),
			Parents: 0,
		}
	} else {
		panic("Cannot prepend a node that doesnt ")
		return nil, errors.New("cannot prepend node with no root")
	}

	// append to map and root keeper
	if idx, ok := t.rootsFasS[head]; ok {
		t.roots[idx] = &newNode
		delete(t.rootsFasS, head)
		t.rootsFasS[&newNode] = idx
	} else {
		lg.Printf("Adding new root: %v", e)
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
func (t *MRootTree) GetRoots() ([]*node) {
	return t.roots[:]
}

// Gets the longest chain, if the length of two chains is exactly the same
// then we return either one of them
func (t *MRootTree) GetLongestChain() *node {
	return t.longestChainHead
}

func New() *MRootTree {
	v := new(MRootTree)
	v.roots = make([]*node, 0, 10)
	v.rootsFasS = make(map[*node]int)
	v.Height = 0

	return v
}