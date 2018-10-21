package block_calculators

import (
	. "../../crypto"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type blkGenList struct {
	addBlockNoop    *int
	addBlockRegular *int
	getRootCalls    *int
	getHighestRoot  *int
	getMinerId      *int
	validate        *int
	blockOps        []*BlockOp
}

func (bg blkGenList) InLongestChain(id string) int {
	return 100
}

func (bg blkGenList) AddBlock(b *Block) {
	fmt.Printf("adding blok %x\n", b.Hash())
	if !b.Valid(numberOfZeros) {
		panic("published invalid block")
	}
	switch b.Type {
	case RegularBlock:
		*bg.addBlockRegular += 1
	case NoOpBlock:
		*bg.addBlockNoop += 1
	}
}

func (bg blkGenList) GetRoots() []*Block {
	*bg.getRootCalls += 1
	return make([]*Block, 0)
}

func (bg blkGenList) GetHighestRoot() *Block {
	*bg.getHighestRoot += 1
	return &Block{
		Type:      GenesisBlock,
		PrevBlock: [md5.Size]byte{12, 1},
	}
}

func (bg blkGenList) GetMinerId() string {
	*bg.getMinerId += 1
	return minerId
}

func (bg blkGenList) ValidateJobSet(bOps []*BlockOp) []*BlockOp {
	*bg.validate += 1
	if len(bOps) == 0 {
		return bOps
	}
	return bg.blockOps
}

const minerId = "william"

var validBlockOps = []*BlockOp{
	{
		Type:         CreateFile,
		RecordNumber: 0,
		Filename:     "beeee",
		Data:         [512]byte{},
		Creator:      minerId,
	},
}

const numberOfZeros = 16

func TestBlockGeneration(t *testing.T) {
	t.Run("generates no-op blocks", func(t *testing.T) {
		listener := blkGenList{
			addBlockNoop:   new(int),
			getMinerId:     new(int),
			getHighestRoot: new(int),
			validate:       new(int),
		}
		bc := NewBlockCalculator(listener, numberOfZeros, 10, 100, 1)
		bc.StartThreads()
		time.Sleep(time.Second)
		bc.ShutdownThreads()

		assert(t, *listener.getHighestRoot > 1, "should have called it to generate blocks")
		assert(t, *listener.addBlockNoop > 5, "should have created at least one block")
	})

	t.Run("adds at least one block", func(t *testing.T) {
		listener := blkGenList{
			addBlockNoop:    new(int),
			addBlockRegular: new(int),
			getMinerId:      new(int),
			getHighestRoot:  new(int),
			validate:        new(int),
			blockOps:        validBlockOps,
		}
		bc := NewBlockCalculator(listener, numberOfZeros, 10, 100, 1)
		bc.StartThreads()
		bc.AddJob(validBlockOps[0])
		time.Sleep(time.Second)
		bc.ShutdownThreads()

		equals(t, 1, *listener.addBlockRegular)
	})

	t.Run("generates three blocks", func(t *testing.T) {
		listener := blkGenList{
			addBlockNoop:    new(int),
			addBlockRegular: new(int),
			getMinerId:      new(int),
			getHighestRoot:  new(int),
			validate:        new(int),
			blockOps:        validBlockOps,
		}
		bc := NewBlockCalculator(listener, numberOfZeros, 10, 100, 1)
		for i := 0; i < 21; i++ {
			bc.AddJob(validBlockOps[0])
		}
		bc.StartThreads()
		time.Sleep(time.Second * 3)
		bc.ShutdownThreads()

		equals(t, 3, *listener.addBlockRegular)
	})

	t.Run("doesn't generate no ops", func(t *testing.T) {
		listener := blkGenList{
			addBlockNoop:    new(int),
			addBlockRegular: new(int),
			getMinerId:      new(int),
			getHighestRoot:  new(int),
			validate:        new(int),
			blockOps:        validBlockOps,
		}
		bc := NewBlockCalculator(listener, numberOfZeros, 10, 100, 1)
		for i := 0; i < 300; i++ {
			bc.AddJob(validBlockOps[0])
		}
		bc.StartThreads()
		time.Sleep(time.Second * 5)
		bc.ShutdownThreads()

		assert(t, *listener.addBlockRegular > 1, "should add 1 more")
		equals(t, 0, *listener.addBlockNoop)
	})

	t.Run("removes repeated jobs when deleting a job", func(t *testing.T) {
		listener := blkGenList{
			addBlockNoop:    new(int),
			addBlockRegular: new(int),
			getMinerId:      new(int),
			getHighestRoot:  new(int),
			validate:        new(int),
			blockOps:        validBlockOps,
		}
		bc := NewBlockCalculator(listener, numberOfZeros, 10, 100, 1)
		for i := 0; i < 300; i++ {
			bc.AddJob(validBlockOps[0])
		}
		bc.RemoveJobsFromBlock(&Block{
			Records: validBlockOps,
		})
		bc.StartThreads()
		time.Sleep(time.Second)
		bc.ShutdownThreads()

		equals(t, 0, *listener.addBlockRegular)
	})
}

// Taken from https://github.com/benbjohnson/testing
// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
