package state

import (
	"../../shared/datastruct"
	"../../crypto"
	"net/rpc"
	"sync"
)

type TreeManager struct {
	clients []*rpc.Client
	mTree *datastruct.MRootTree
	findBlockQueue *datastruct.Queue
	mtx *sync.Mutex
}

func (t *TreeManager) AddClient(c *rpc.Client) {
	t.clients = append(t.clients, c)
}


func (t *TreeManager) AddBlock(b *crypto.BlockElement) error {
	if blk, ok := t.mTree.Find(b.ParentId()); ok {
		// simple case: the reference node is in the chain
		_, err := t.mTree.PrependElement(b, blk)
		return err
	}
	if b.Block.Type == crypto.GenesisBlock {
		// second case, its the genesis case
		_, err := t.mTree.PrependElement(b, nil)
		return err
	}

	eqFn := func(e datastruct.QueueElement) bool {
		bl := e.(*crypto.BlockElement)
		return bl.Id() == b.Id()
	}

	if t.findBlockQueue.IsInQueue(eqFn) {
		// third case, the block is in the queue so don't do anything
		return nil
	}

	// harder case.. we don't know where this block came from,
	// queue it and defer it to a another place
	t.findBlockQueue.Enqueue(b)
	return nil
}

func (t *TreeManager) GetLongestChain() *datastruct.Node {
	return t.mTree.GetLongestChain()
}

// TODO vpineda create 2 go routines
// TODO 1 go routine for the the queue that will call // flood rpc endpoints to ask for nodes
// TODO 1 go routine to every now and then sync roots with other peers

func NewTreeManager() *TreeManager {
	return &TreeManager{
		mtx:     new(sync.Mutex),
		clients: make([]*rpc.Client, 0, 1),
		mTree:   datastruct.NewMRootTree(),
	}
}