package state

import (
	"../../shared/datastruct"
	"../../crypto"
	"net/rpc"
	"sync"
)

// This is one of the most critical areas of a miner, it is the only one that will have access to
// the blockchain tree itself
type TreeManager struct {
	clients []*rpc.Client
	mTree BlockChainTree
	findBlockQueue *datastruct.Queue
	mtx *sync.Mutex
}

func (t *TreeManager) AddClient(c *rpc.Client) {
	t.clients = append(t.clients, c)
}

func (t *TreeManager) GetBlock(id string) (*crypto.Block, bool){
	v, ok := t.mTree.Find(id)
	if !ok {
		return nil, false
	}
	bk, ok := v.Value.(crypto.BlockElement)
	if !ok {
		return nil, false
	}
	return bk.Block, true
}

func (t *TreeManager) GetRoots() []*crypto.Block {
	arr := t.mTree.GetRoots()
	bkArr := make([]*crypto.Block, len(arr))
	for i, v := range arr {
		bk, ok := v.Value.(crypto.BlockElement)
		if !ok {
			continue
		}
		bkArr[i] = bk.Block
	}
	return bkArr
}

func (t *TreeManager) AddBlock(b crypto.BlockElement) error {
	if _, ok := t.mTree.Find(b.ParentId()); ok {
		// simple case: the reference node is in the chain
		_, err := t.mTree.Add(b)
		if err != nil {
			return err
		}
	}
	if b.Block.Type == crypto.GenesisBlock {
		// second case, its the genesis case
		_, err := t.mTree.Add(b)
		if err != nil {
			return err
		}
	}

	eqFn := func(e datastruct.QueueElement) bool {
		bl := e.(crypto.BlockElement)
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


func NewTreeManager(cnf Config) *TreeManager {
	tree := datastruct.NewMRootTree()
	return &TreeManager{
		mtx:     new(sync.Mutex),
		clients: make([]*rpc.Client, 0, 1),
		mTree:   BlockChainTree{
			mTree: tree,
			validator: NewBlockChainValidator(cnf, tree),
		},
		findBlockQueue: &datastruct.Queue{},
	}
}

type BlockChainTree struct {
	mTree *datastruct.MRootTree
	validator *BlockChainValidator
}

func (b BlockChainTree) Find(id string) (*datastruct.Node, bool){
	return b.mTree.Find(id)
}

// adds block to the blockchain give that it passes all validations
func (b BlockChainTree) Add(block crypto.BlockElement) (*datastruct.Node, error) {
	root, err := b.validator.Validate(block)
	if err != nil {
		lg.Printf("Rejected block %v, due to %v\n", block.Id(), err)
		return nil, err
	}
	lg.Printf("Added block %v\n", block.Id())
	return b.mTree.PrependElement(block, root)
}

func (b BlockChainTree) GetLongestChain() *datastruct.Node {
	return b.mTree.GetLongestChain()
}

func (b BlockChainTree) GetRoots() []*datastruct.Node{
	return b.mTree.GetRoots()
}