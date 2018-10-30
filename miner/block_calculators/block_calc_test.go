package block_calculators

import (
	. "../../crypto"
	"crypto/md5"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
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
	longestNum      *int
	blockOps        []*BlockOp
}

func (bg blkGenList) InLongestChain(id string) int {
	if bg.longestNum != nil {
		return *bg.longestNum
	}
	return 100
}

func (bg blkGenList) AddBlock(b *Block) {
	fmt.Printf("adding blok %x\n", b.Hash())
	if !b.Valid(numberOfZeros, numberOfZeros) {
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

func (bg blkGenList) ValidateJobSet(bOps []*BlockOp) ([]*BlockOp, error, error) {
	*bg.validate += 1
	if len(bOps) == 0 {
		return bOps, nil, nil
	}
	return bg.blockOps, nil, nil
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

const numberOfZeros = 4

func TestBlockGeneration(t *testing.T) {
	t.Run("generates no-op blocks", func(t *testing.T) {
		listener := blkGenList{
			addBlockNoop:   new(int),
			getMinerId:     new(int),
			getHighestRoot: new(int),
			validate:       new(int),
		}
		bc := NewBlockCalculator(listener, numberOfZeros, numberOfZeros, 10, 100, 1)
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
		bc := NewBlockCalculator(listener, numberOfZeros, numberOfZeros,10, 100, 1)
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
		bc := NewBlockCalculator(listener, numberOfZeros, numberOfZeros,10, 100, 1)
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
		bc := NewBlockCalculator(listener, numberOfZeros, numberOfZeros,10, 100, -1)
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
		bc := NewBlockCalculator(listener, numberOfZeros, numberOfZeros,10, 100, 1)
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

	t.Run("waits for ops and packages them according to timeout", func(t *testing.T) {
		listener := blkGenList{
			addBlockNoop:    new(int),
			addBlockRegular: new(int),
			getMinerId:      new(int),
			getHighestRoot:  new(int),
			validate:        new(int),
			blockOps:        validBlockOps,
		}
		bc := NewBlockCalculator(listener, numberOfZeros, numberOfZeros,10, 500, 1)

		for i := 0; i < 2; i++ {
			bop := validBlockOps[0]
			bop.Creator = strconv.Itoa(rand.Int())
			bc.AddJob(bop)
		}
		bc.StartThreads()
		time.Sleep(time.Millisecond * 100)
		for i := 0; i < 18; i++ {
			bop := validBlockOps[0]
			bop.Creator = strconv.Itoa(rand.Int())
			bc.AddJob(bop)
		}
		time.Sleep(time.Second * 3)
		bc.ShutdownThreads()

		equals(t, 2, *listener.addBlockRegular)
	})

	t.Run("keeps trying the same job if it doesn't belong to the longest chain", func(t *testing.T) {
		lgInt := -1 // not in longest chain
		listener := blkGenList{
			addBlockNoop:    new(int),
			addBlockRegular: new(int),
			getMinerId:      new(int),
			getHighestRoot:  new(int),
			validate:        new(int),
			longestNum:      &lgInt,
			blockOps:        validBlockOps,
		}
		bc := NewBlockCalculator(listener, numberOfZeros, numberOfZeros,10, 500, 10)

		for i := 0; i < 2; i++ {
			bop := validBlockOps[0]
			bop.Creator = strconv.Itoa(rand.Int())
			bc.AddJob(bop)
		}
		bc.StartThreads()
		time.Sleep(time.Second * 3)
		assert(t, *listener.addBlockRegular > 2, "if this is false then we are not trying to " +
			"add blocks that werent added to longest chain")
		lgInt = 200
		currentC := *listener.addBlockRegular
		// now there is enough depth wait for mining to finish
		time.Sleep(time.Second)
		// only one more call to actually add the block
		equals(t, currentC + 1, *listener.addBlockRegular)
		bc.ShutdownThreads()
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
