package block_calculators

import (
	"../../crypto"
	"../../shared/datastruct"
	"bytes"
	"container/heap"
	"crypto/md5"
	"log"
	"os"
	"reflect"
	"sync"
	"time"
)

type BlockCalculatorListener interface {
	AddBlock(b *crypto.Block)
	GetRoots() []*crypto.Block
	GetHighestRoot() *crypto.Block
	GetMinerId() string
	ValidateJobSet(bOps []*crypto.BlockOp) []*crypto.BlockOp
}

type BlockCalculator struct {
	listener        BlockCalculatorListener
	jobSet          *datastruct.PriorityQueue
	noopSuspended   bool
	shutdownThreads bool
	mtx             *sync.Mutex
	opsPerBlock     int
	numberOfZeros   int
	timePerBlockTimeoutMillis time.Duration
}

var lg = log.New(os.Stdout, "calculators: ", log.Lmicroseconds|log.Lshortfile)

func (bc *BlockCalculator) AddJob(b *crypto.BlockOp) {
	bc.noopSuspended = true
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	item := datastruct.Item{
		Value: b,
	}
	heap.Push(bc.jobSet, &item)
}


func (bc *BlockCalculator) RemoveJobsFromBlock(block *crypto.Block) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	for _, rc := range block.Records {
		eqFn := func(j interface{}) bool {
			job := j.(*crypto.BlockOp)
			return reflect.DeepEqual(*job, *rc)
		}
		for hpIdx := bc.jobSet.Find(eqFn); hpIdx >=0; hpIdx = bc.jobSet.Find(eqFn) {
			heap.Remove(bc.jobSet, hpIdx)
		}
	}
}

func (bc *BlockCalculator) ShutdownThreads() {
	bc.shutdownThreads = true
}

func (bc *BlockCalculator) StartThreads() {
	bc.shutdownThreads = false
	go NoOpCalculator(bc)
	go JobsCalculator(bc)
}

func NoOpCalculator(bc *BlockCalculator) {
	for !bc.shutdownThreads {
		newBlock := generateNewBlock(bc, []*crypto.BlockOp{}, &bc.noopSuspended)
		if !bc.noopSuspended && bytes.Equal(bc.listener.GetHighestRoot().Hash(), newBlock.PrevBlock[:]) {
			lg.Printf("No-op calculator found a block")
			bc.listener.AddBlock(newBlock)
		}
		time.Sleep(time.Millisecond * bc.timePerBlockTimeoutMillis)
	}
}
func generateNewBlock(bc *BlockCalculator, ops []*crypto.BlockOp, suspendBool *bool) *crypto.Block {
	rootHash := [md5.Size]byte{}
	copy(rootHash[:], bc.listener.GetHighestRoot().Hash())

	bk := crypto.Block{
		MinerId: bc.listener.GetMinerId(),
		Type: crypto.NoOpBlock,
		Nonce: 0,
		Records: ops,
		PrevBlock: rootHash,
	}
	bk.FindNonceWithStopSignal(bc.numberOfZeros, suspendBool)
	return &bk
}

func JobsCalculator(bc *BlockCalculator) {
	for !bc.shutdownThreads {
		blockOps := getBlockOps(bc)
		if len(blockOps) > 0 {
			// stop noop thread and start mining your own block
			bc.noopSuspended = true
			newBlock := generateNewBlock(bc, blockOps, new(bool))
			newBlock.Type = crypto.RegularBlock

			// once we found a block send it and remove those jobs form the queue
			if bytes.Equal(bc.listener.GetHighestRoot().Hash(), newBlock.PrevBlock[:]) {
				lg.Printf("Jobs calculator found a block")
				bc.listener.AddBlock(newBlock)
			}
		} else {
			bc.noopSuspended = false
		}

		time.Sleep(time.Millisecond * bc.timePerBlockTimeoutMillis)
	}
}

func getBlockOps(bc *BlockCalculator) []*crypto.BlockOp {
	bc.mtx.Lock()
	bOps := make([]*crypto.BlockOp, 0, bc.opsPerBlock)
	for i := 0; i < bc.opsPerBlock && bc.jobSet.Len() > 0; i++ {
		blk := heap.Pop(bc.jobSet).(*datastruct.Item).Value.(*crypto.BlockOp)
		bOps = append(bOps, blk)
	}
	bc.mtx.Unlock()
	return bc.listener.ValidateJobSet(bOps)
}


func NewBlockCalculator(state BlockCalculatorListener, numberOfZeros int, opsPerBlock int, blockTimeout time.Duration) *BlockCalculator {
	return &BlockCalculator{
		jobSet:      new(datastruct.PriorityQueue),
		listener:    state,
		mtx:         new(sync.Mutex),
		numberOfZeros: numberOfZeros,
		opsPerBlock: opsPerBlock,
		timePerBlockTimeoutMillis: blockTimeout,
	}
}