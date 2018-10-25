package state

import (
	"../../crypto"
	"../../shared/datastruct"
	"sync"
	"time"
)

type BlockRetriever interface {
	// gets remote block and validates that the id that was given is equal to the cryptoblock
	GetRemoteBlock(id string) (*crypto.Block, bool)
	GetRemoteRoots() []*crypto.Block
}

type TreeChangeListener interface {
	OnNewBlockInTree(b *crypto.Block)
	OnNewBlockInLongestChain(b *crypto.Block)
}

// This is one of the most critical areas of a miner, it is the only one that will have access to
// the blockchain tree itself
type TreeManager struct {
	br              BlockRetriever
	MTree           BlockChainTree
	findBlockQueue  *datastruct.Queue
	findBlockNotify chan bool
	shutdownThreads bool
}

func (t *TreeManager) GetBlock(id string) (*crypto.Block, bool) {
	v, ok := t.MTree.Find(id)
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
	arr := t.MTree.GetRoots()
	bkArr := make([]*crypto.Block, len(arr))
	for i, v := range arr {
		bk, ok := v.Value.(crypto.BlockElement)
		if !ok {
			continue
		}
		cpy := *bk.Block
		bkArr[i] = &cpy
	}
	return bkArr
}

func (t *TreeManager) AddBlock(b crypto.BlockElement) error {
	if _, ok := t.MTree.Find(b.ParentId()); ok {
		// simple case: the reference node is in the chain
		if _, ok := t.MTree.Find(b.Id()); !ok {
			_, err := t.MTree.Add(b)
			if err != nil {
				return err
			}
		}
		return nil
	}
	if b.Block.Type == crypto.GenesisBlock {
		// second case, its the genesis case

		_, err := t.MTree.Add(b)
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
	lg.Printf("Enqueuing block %v\n", b.Id())
	t.findBlockQueue.Enqueue(b)
	select {
	case t.findBlockNotify <- true:
	default:
	}
	return nil
}

func (t *TreeManager) Exists(b *crypto.Block) bool {
	_, exists := t.MTree.Find(b.Id())
	return exists
}

func (t *TreeManager) GetHighestRoot() *crypto.Block {
	cpy := *t.MTree.GetLongestChain().Value.(crypto.BlockElement).Block
	return &cpy
}

func (t *TreeManager) InLongestChain(id string) int {
	return t.MTree.InLongestChain(id)
}

func (t *TreeManager) ValidateBlock(b *crypto.Block) bool {
	return t.MTree.ValidateBlock(b)
}

func (t *TreeManager) GetLongestChain() *datastruct.Node {
	return t.MTree.GetLongestChain()
}

func (t *TreeManager) ValidateJobSet(bOps []*crypto.BlockOp) []*crypto.BlockOp {
	return t.MTree.ValidateJobSet(bOps)
}

func (t *TreeManager) ShutdownThreads() {
	t.shutdownThreads = true
}

func (t *TreeManager) StartThreads() {
	t.shutdownThreads = false
	go FindNodeThread(t)
	go UpdateRootsThread(t)
}

func blockAdderHelper(t *TreeManager, b crypto.BlockElement) bool {
	// the block is a genesis block no need to check children add it directly
	if b.Block.Type == crypto.GenesisBlock {
		err := t.AddBlock(b)
		if err != nil {
			removeNodesStartingFrom(b, t.findBlockQueue)
			return false
		}
		return true
	}

	// parent block is in the tree, add it
	if _, ok := t.MTree.Find(b.ParentId()); ok {
		if _, ok := t.MTree.Find(b.Id()); !ok {
			err := t.AddBlock(b)
			if err != nil {
				removeNodesStartingFrom(b, t.findBlockQueue)
				return false
			}
		}
		return true
	}

	// parent block is in the queue, enqueue
	eqParIdFn := func(e datastruct.QueueElement) bool {
		bl := e.(crypto.BlockElement)
		return bl.Id() == b.ParentId()
	}

	if t.findBlockQueue.IsInQueue(eqParIdFn) {
		lg.Printf("Found parent %v, in queue. Queuing block %v", b.ParentId(), b.Id())
		err := t.AddBlock(b)
		if err != nil {
			removeNodesStartingFrom(b, t.findBlockQueue)
			return false
		}
		return true
	}

	// parent block needs to be searched, and then added to tree
	block, ok := t.br.GetRemoteBlock(b.ParentId())
	if !ok {
		lg.Printf("Discarding block %v since no node knows about its parent", b.Id())
		removeNodesStartingFrom(b, t.findBlockQueue)
		return false
	} else {
		// we found the parent block!, add the parent if the parent of the parent is there
		if _, ok := t.MTree.Find(b.ParentId()); ok {
			lg.Printf("Found parent that is viable adding")
			err := t.AddBlock(crypto.BlockElement{
				Block: block,
			})
			if err == nil {
				err := t.AddBlock(b)
				if err != nil {
					removeNodesStartingFrom(b, t.findBlockQueue)
					return false
				}
				return true
			} else {
				// remove all of the nodes that had b as parent
				removeNodesStartingFrom(b, t.findBlockQueue)
				return false
			}
		} else {
			success := blockAdderHelper(t, crypto.BlockElement{
				Block: block,
			})
			if success {
				lg.Printf("Adding missing block %v", b.Id())
				t.AddBlock(b)
			}
			return success
		}
	}
}

func removeNodesStartingFrom(block crypto.BlockElement, q *datastruct.Queue) bool {
	parentId := block.Id()
	fDeleteSearch := func(e datastruct.QueueElement) bool {
		bl := e.(crypto.BlockElement)
		if bl.ParentId() == parentId {
			parentId = bl.Id()
			lg.Printf("Deleting block %v because is on an illegal chain of %v", bl.Id(), block.Id())
			return true
		}
		return false
	}
	for q.Del(fDeleteSearch) {
	}
	return false
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
		time.Sleep(time.Second)
	}
}

func NewTreeManager(cnf Config, br BlockRetriever, tcl TreeChangeListener) *TreeManager {
	tree := datastruct.NewMRootTree()
	tm := &TreeManager{
		br:  br,
		MTree: BlockChainTree{
			mTree:     tree,
			tcl:       tcl,
			mtx:       new(sync.Mutex),
			Validator: NewBlockChainValidator(cnf, tree),
		},
		findBlockQueue:  &datastruct.Queue{},
		findBlockNotify: make(chan bool),
		shutdownThreads: false,
	}
	return tm
}

type BlockChainTree struct {
	mTree     *datastruct.MRootTree
	Validator *BlockChainValidator
	tcl       TreeChangeListener
	mtx       *sync.Mutex
}

func (b BlockChainTree) Find(id string) (*datastruct.Node, bool) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.mTree.Find(id)
}

// adds block to the blockchain give that it passes all validations
func (b BlockChainTree) Add(block crypto.BlockElement) (*datastruct.Node, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	root, err := b.Validator.Validate(block)
	if err != nil {
		lg.Printf("Rejected block %v, due to %v\n", block.Id(), err)
		return nil, err
	}

	nd, err := b.mTree.PrependElement(block, root)
	lg.Printf("Added block %v\n", block.Id())
	if err != nil {
		return nil, err
	}

	go b.tcl.OnNewBlockInTree(block.Block)
	if b.GetLongestChain().Id == block.Id() {
		go b.tcl.OnNewBlockInLongestChain(block.Block)
	}

	return nd, err
}

func (b BlockChainTree) InLongestChain(id string) int {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	depth := 0
	for r := b.GetLongestChain(); r != nil; r = r.Next() {
		if r.Id == id {
			return depth
		}
		depth += 1
	}
	return -1
}

func (b BlockChainTree) GetLongestChain() *datastruct.Node {
	return b.mTree.GetLongestChain()
}

func (b BlockChainTree) ValidateBlock(blk *crypto.Block) bool {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	_, err := b.Validator.Validate(crypto.BlockElement{
		Block: blk,
	})
	return err != nil
}

func (b BlockChainTree) GetRoots() []*datastruct.Node {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.mTree.GetRoots()
}

func (b BlockChainTree) ValidateJobSet(ops []*crypto.BlockOp) []*crypto.BlockOp {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.Validator.ValidateJobSet(ops, b.mTree.GetLongestChain())
}
