package block_calculators

import (
	"../../crypto"
	"bytes"
	"crypto/md5"
	"sync"
	"time"
)

type BlockCalculatorListener interface {
	AddBlock(b *crypto.Block)
	GetRoots() []*crypto.Block
	GetHighestRoot() *crypto.Block
	GetTargetZeros() int
	GetMinerId() string
	ValidateBlock(b *crypto.Block) bool
}

type BlockCalculator struct {
	listener        BlockCalculatorListener
	jobSet          *[]*crypto.BlockOp
	noopSuspended   bool
	shutdownThreads bool
	mtx             *sync.Mutex
	opsPerBlock     int
}

func (bc *BlockCalculator) AddJob(b *crypto.BlockOp) {
	panic("implement me")
}

func (bc *BlockCalculator) ShutdownThreads() {
	bc.shutdownThreads = true
}

func NoOpCalculator(bc *BlockCalculator) {
	for !bc.shutdownThreads {
		newBlock := generateNewBlock(bc, []*crypto.BlockOp{}, &bc.noopSuspended)
		if !bc.noopSuspended && bytes.Equal(bc.listener.GetHighestRoot().Hash(), newBlock.PrevBlock[:]) {
			bc.listener.AddBlock(newBlock)
		}
		time.Sleep(time.Second)
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
	bk.FindNonceWithStopSignal(bc.listener.GetTargetZeros(), suspendBool)
	return &bk
}

func JobsCalculator(bc *BlockCalculator) {
	for !bc.shutdownThreads {
		blockOps := getBlockOps(bc)
		if len(blockOps) > 0 {
			// stop noop thread and start mining your own block
			bc.noopSuspended = true
			newBlock := generateNewBlock(bc, blockOps, new(bool))

			// once we found a block send it and remove those jobs form the queue
			if bytes.Equal(bc.listener.GetHighestRoot().Hash(), newBlock.PrevBlock[:]) {
				bc.listener.AddBlock(newBlock)
				updateQueue(bc, newBlock)
			}
		} else {
			bc.noopSuspended = false
		}

		time.Sleep(time.Second)
	}
}
func updateQueue(bc *BlockCalculator, block *crypto.Block) {
	// first remove the records on the block
	for _, v := range block.Records {
		for i, bPtr := range *bc.jobSet {
			if v.Filename == bPtr.Filename &&
				v.Type == bPtr.Type &&
				(v.Type != crypto.AppendFile || v.RecordNumber >= bPtr.RecordNumber){
					if i == len(*bc.jobSet) - 1 {
						*bc.jobSet = (*bc.jobSet)[:i]
					} else {
						*bc.jobSet = append((*bc.jobSet)[:i], (*bc.jobSet)[i+1:]...)
					}
					break
			}
		}
	}
	// create a fs state and validate that the jobs that we are working on are
	// valid
}

func getBlockOps(bc *BlockCalculator) []*crypto.BlockOp {
	if len(*bc.jobSet) < bc.opsPerBlock {
		return (*bc.jobSet)[:]
	} else {
		return (*bc.jobSet)[:bc.opsPerBlock - 1]
	}
}


func NewBlockCalculator(state BlockCalculatorListener, opsPerBlock int) *BlockCalculator {
	return &BlockCalculator{
		jobSet:      new([]*crypto.BlockOp),
		listener:    state,
		mtx:         new(sync.Mutex),
		opsPerBlock: opsPerBlock,
	}
}