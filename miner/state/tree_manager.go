package state

import (
	"../../crypto"
	"../../shared/datastruct"
	"sync"
	"time"
)


type BlockRetriever interface {
	GetRemoteBlock(id string) (*crypto.Block, bool)
	GetRemoteRoots() ([]*crypto.Block)
}

// This is one of the most critical areas of a miner, it is the only one that will have access to
// the blockchain tree itself
type TreeManager struct {
	br BlockRetriever
	mTree BlockChainTree
	findBlockQueue *datastruct.Queue
	findBlockNotify chan bool
	shutdownThreads bool
	mtx *sync.Mutex
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
		if _, ok := t.mTree.Find(b.Id()); !ok {
			_, err := t.mTree.Add(b)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if b.Block.Type == crypto.GenesisBlock {
		// second case, its the genesis case
		_, err := t.mTree.Add(b)
		if err != nil {
			return err
		}
		return nil
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
	select {
	case t.findBlockNotify <- true:
	default:
	}
	return nil
}

func (t *TreeManager) GetLongestChain() *datastruct.Node {
	return t.mTree.GetLongestChain()
}

// TODO vpineda create 2 go routines
// TODO 1 go routine for the the queue that will call  // flood rpc endpoints to ask for nodes
// TODO 1 go routine to every now and then sync roots with other peers

func blockAdderHelper(t* TreeManager, b crypto.BlockElement) bool {
	// parent block is in the tree, add it
	if _, ok := t.mTree.Find(b.ParentId()); ok {
		if _, ok := t.mTree.Find(b.Id()); ok {

		}
		t.AddBlock(b)
		return true
	}

	// parent block is in the queue, enqueue
	eqParIdFn := func(e datastruct.QueueElement) bool {
		bl := e.(crypto.BlockElement)
		return bl.ParentId() == b.ParentId()
	}

	if t.findBlockQueue.IsInQueue(eqParIdFn) {
		t.AddBlock(b)
		return true
	}

	// parent block needs to be searched, and then added to tree
	block, ok := t.br.GetRemoteBlock(b.ParentId())
	if !ok {
		lg.Printf("Discarding block %v since no node knows about its parent", b.Id())
		return false
	} else {
		// we found the parent block!, add the parent to the queue and then the child
		t.AddBlock(crypto.BlockElement{
			Block: block,
		})
		t.AddBlock(b)
		return true
	}
}

func FindNodeThread(t *TreeManager) {
	for !t.shutdownThreads {
		for v, ok := t.findBlockQueue.Dequeue(); ok; v, ok = t.findBlockQueue.Dequeue() {
			block := v.(crypto.BlockElement)
			blockAdderHelper(t, block)
		}
		select {
		case <-t.findBlockNotify:
		}
	}
}

func UpdateRootsThread(t *TreeManager) {
	for !t.shutdownThreads {
		roots := t.br.GetRemoteRoots()
		for _, block := range roots {
			blockAdderHelper(t, crypto.BlockElement{
				Block: block,
			})
		}
		time.Sleep(time.Second * 10)
	}
}


func NewTreeManager(cnf Config, br BlockRetriever) *TreeManager {
	tree := datastruct.NewMRootTree()
	tm :=  &TreeManager{
		mtx:     new(sync.Mutex),
		br: br,
		mTree:   BlockChainTree{
			mTree: tree,
			validator: NewBlockChainValidator(cnf, tree),
		},
		findBlockQueue: &datastruct.Queue{},
		findBlockNotify: make(chan bool),
		shutdownThreads: false,
	}
	go FindNodeThread(tm)
	go UpdateRootsThread(tm)
	return tm
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