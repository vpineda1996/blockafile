package block_calculators

import (
	"../../crypto"
	"../../shared/datastruct"
	"bytes"
	"container/heap"
	"crypto/md5"
	"io/ioutil"
	"log"
	"math"
	"reflect"
	"sync"
	"time"
)

type BlockCalculatorListener interface {
	AddBlock(b *crypto.Block)
	GetRoots() []*crypto.Block
	GetHighestRoot() *crypto.Block
	GetMinerId() string
	ValidateJobSet(bOps []*crypto.BlockOp) ([]*crypto.BlockOp, error, error)
	InLongestChain(id string) int
}

type BlockCalculator struct {
	listener                  BlockCalculatorListener
	jobSet                    *datastruct.PriorityQueue
	noopSuspended             bool
	opSuspended               bool
	shutdownThreads           bool
	mtx                       *sync.Mutex
	opsPerBlock               int
	opNumberOfZeros           int
	noOpNumberOfZeros         int
	maxConfirm				  int
	timePerBlockTimeoutMillis time.Duration
}

var lg = log.New(ioutil.Discard, "calculators: ", log.Lmicroseconds|log.Lshortfile)
var counter = math.MaxInt32

func (bc *BlockCalculator) AddJob(b *crypto.BlockOp) {
	bc.noopSuspended = true
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	item := datastruct.Item{
		Value:    b,
		Priority: counter,
	}
	counter -= 1
	heap.Push(bc.jobSet, &item)
}

func (bc *BlockCalculator) JobExists(b *crypto.BlockOp) int {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	return bc.jobExists(b)
}
func (bc *BlockCalculator) jobExists(b *crypto.BlockOp) int {
	eqFn := func(j interface{}) bool {
		job := j.(*crypto.BlockOp)
		return reflect.DeepEqual(*job, *b)
	}
	return bc.jobSet.Find(eqFn)
}

func (bc *BlockCalculator) RemoveJobsFromBlock(block *crypto.Block) {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	for _, rc := range block.Records {
		for hpIdx := bc.jobExists(rc); hpIdx >= 0; hpIdx = bc.jobExists(rc) {
			heap.Remove(bc.jobSet, hpIdx)
		}
	}
	bc.opSuspended = true
	bc.noopSuspended = true
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
		newBlock := generateNewBlock(bc, []*crypto.BlockOp{}, &bc.noopSuspended, crypto.NoOpBlock)
		if !bc.noopSuspended && bytes.Equal(bc.listener.GetHighestRoot().Hash(), newBlock.PrevBlock[:]) {
			bc.listener.AddBlock(newBlock)
		}
		time.Sleep(time.Millisecond * 50)
	}
}
func generateNewBlock(bc *BlockCalculator, ops []*crypto.BlockOp, suspendBool *bool, blockType crypto.BlockType) *crypto.Block {
	rootHash := [md5.Size]byte{}
	copy(rootHash[:], bc.listener.GetHighestRoot().Hash())

	bk := crypto.Block{
		MinerId:   bc.listener.GetMinerId(),
		Type:      blockType,
		Nonce:     0,
		Records:   ops,
		PrevBlock: rootHash,
	}

	zeros := bk.GetZerosForType(bc.opNumberOfZeros, bc.noOpNumberOfZeros)
	bk.FindNonceWithStopSignal(zeros, suspendBool)
	return &bk
}

func addedToLongestChainValidation(bc *BlockCalculator, block *crypto.Block) bool {
	defer func() {bc.noopSuspended = true}()
	for {
		depth := bc.listener.InLongestChain(block.Id())
		if depth < 0 {
			return false
		} else if depth > bc.maxConfirm {
			return true
		}
		bc.noopSuspended = false
		time.Sleep(time.Millisecond * 50)
	}
}

func JobsCalculator(bc *BlockCalculator) {
	for !bc.shutdownThreads {
		blockOps := getBlockOps(bc)
		if len(blockOps) > 0 {
			// stop noop thread and start mining your own block
			bc.noopSuspended = true
			for {
				bc.opSuspended = false
				newBlock := generateNewBlock(bc, blockOps, &bc.opSuspended, crypto.RegularBlock)
				lg.Printf("Generated block with %v ops", len(blockOps))
				// once we found a block send it and remove those jobs form the queue
				if bytes.Equal(bc.listener.GetHighestRoot().Hash(), newBlock.PrevBlock[:]) {
					lg.Printf("Jobs calculator found a block")
					bc.listener.AddBlock(newBlock)

					if !addedToLongestChainValidation(bc, newBlock) {
						// re-enqueue jobs if we didn't add and start from scratch
						lg.Printf("Block wasn't added to blockchain, putting it on the backburner")
						for _, r := range newBlock.Records {
							bc.AddJob(r)
						}
					}
					break
				} else if bc.opSuspended {
					// if the op was suspended, retry doing the job again, worst case we filter out the op
					// when its repeated
					for _, r := range newBlock.Records {
						bc.AddJob(r)
					}
					break
				}
			}
		} else {
			bc.noopSuspended = false
		}

		time.Sleep(time.Millisecond * 50)
	}
}

func getBlockOps(bc *BlockCalculator) []*crypto.BlockOp {
	bc.mtx.Lock()
	defer bc.mtx.Unlock()
	bOps := make([]*crypto.BlockOp, 0, bc.opsPerBlock)
	for i := 0; i < (bc.opsPerBlock + 1) && bc.jobSet.Len() > 0; i++ {
		if i == 0 {
			bc.mtx.Unlock()
			time.Sleep(time.Millisecond * bc.timePerBlockTimeoutMillis)
			bc.mtx.Lock()
		} else {
			blk := heap.Pop(bc.jobSet).(*datastruct.Item).Value.(*crypto.BlockOp)
			bOps = append(bOps, blk)
		}
	}

	newOps, _, _ := bc.listener.ValidateJobSet(bOps)
	return newOps
}

func NewBlockCalculator(state BlockCalculatorListener,
	opNumberOfZeros int,
	noOpNumberOfZeros int,
	opsPerBlock int,
	blockTimeout time.Duration, maxConfirm int) *BlockCalculator {
	bc := &BlockCalculator{
		jobSet:                    new(datastruct.PriorityQueue),
		listener:                  state,
		mtx:                       new(sync.Mutex),
		opNumberOfZeros:           opNumberOfZeros,
		noOpNumberOfZeros:		   noOpNumberOfZeros,
		opsPerBlock:               opsPerBlock,
		timePerBlockTimeoutMillis: blockTimeout,
		maxConfirm: maxConfirm,
	}
	heap.Init(bc.jobSet)
	return bc
}
