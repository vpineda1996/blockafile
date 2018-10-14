package state

import (
	"../../crypto"
	"crypto/md5"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
)
import (
	. "../../shared/datastruct"
)

type treeBuilderTest struct {
	height uint64
	roots int
	addOrder []int   // node where we insert first | the number of nodes we insert | id of miner | tpe | txs | id of creator
}

var genBlockSeed = [md5.Size]byte{10, 2,1 }
var blockReward = 1
var txFee = 1

var counter byte = 0

func buildTree(treeDef treeBuilderTest) *MRootTree {

	test := treeDef
	nds := make([]*Node, 0, 100)
	ee := crypto.BlockElement{
		Block: &crypto.Block {
			MinerId: strconv.Itoa(1),
			Type: crypto.GenesisBlock,
			PrevBlock: genBlockSeed,
			Records: []*crypto.BlockOp{},
			Nonce: 12324,
		},
	}
	mtr := NewMRootTree()

	// create a root
	e, _ :=  mtr.PrependElement(ee, nil)
	nds = append(nds, e)

	for i := 0; i < len(test.addOrder); i+= 6 {
		// grab root and start adding n nodes
		root := nds[test.addOrder[i]]
		for j := 0; j < test.addOrder[i+1]; j++ {
			records := make([]*crypto.BlockOp, test.addOrder[i+4])
			for u := 0; u < test.addOrder[i+4]; u++ {
				record := crypto.BlockOp{
					Type: crypto.CreateFile,
					Filename: "random" + strconv.Itoa(rand.Int()),
					Data: [crypto.DataBlockSize]byte{counter},
					Creator: strconv.Itoa(test.addOrder[i+5]),
				}
				records[u] = &record
				counter += 1
			}
			prevBlk := [md5.Size]byte{}
			copy(prevBlk[:], root.Value.(crypto.BlockElement).Block.Hash())
			ee := crypto.BlockElement{
				Block: &crypto.Block {
					MinerId: strconv.Itoa(test.addOrder[i+2]),
					Type: crypto.BlockType(test.addOrder[i+3]),
					PrevBlock: prevBlk,
					Records: records,
					Nonce: 12324,
				},
			}
			var err error
			root, err = mtr.PrependElement(ee, root)
			if err != nil {
				panic(err)
			}
			nds = append(nds, root)
		}
	}
	return mtr

}

func TestSimpleBlockChainTree(t *testing.T) {
	t.Run("returns empty state on empty tree", func(t *testing.T) {
		tree := NewMRootTree()
		bkState, _ := NewAccountsState(blockReward, txFee, tree.GetLongestChain())
		equals(t, 0, len(bkState.GetAll()))
	})

	t.Run("simple tree with just the genesis block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{},
		}
		tree := buildTree(treeDef)

		bkState, _ := NewAccountsState(blockReward, txFee, tree.GetLongestChain())
		equals(t, 0, len(bkState.GetAll()))
	})

	t.Run("simple tree with just genesis and a no-op block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 1, 1, int(crypto.NoOpBlock), 0, 1},
		}
		tree := buildTree(treeDef)
		bkState, _ := NewAccountsState(blockReward, txFee, tree.GetLongestChain())
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 1
		equals(t, mp, bkState.GetAll())
	})

	t.Run("simple tree with just genesis, no-op block, op block with 1 tx", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots: 1,
			addOrder: []int{
				0, 1, 1, int(crypto.NoOpBlock), 0, 1,
				1, 1, 2, int(crypto.RegularBlock), 1, 1},
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(blockReward, txFee, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 0
		mp[Account(strconv.Itoa(2))] = 1
		equals(t, mp, bkState.GetAll())
	})
}

func TestComplexBlockChainTree(t *testing.T) {
	t.Run("long branch with multiple accounts", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots: 1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1},
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(blockReward, txFee, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 100
		mp[Account(strconv.Itoa(2))] = 1
		mp[Account(strconv.Itoa(3))] = 3
		equals(t, mp, bkState.GetAll())
	})

	t.Run("it only follows tnx on longest chain", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots: 1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,

				// fake chain
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,

				// longest chain
				100, 1, 5, int(crypto.NoOpBlock), 0, 1,     // id: 105
				105, 30, 2, int(crypto.NoOpBlock), 0, 1},   // id: 106
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(blockReward, txFee, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 100
		mp[Account(strconv.Itoa(2))] = 30
		mp[Account(strconv.Itoa(5))] = 1
		equals(t, mp, bkState.GetAll())
	})

	t.Run("FUN stands for fucked under necessary conditions", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots: 1,
			addOrder: []int{
				// first part chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,

				// fake chain
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,

				// middle chain
				100, 1, 5, int(crypto.NoOpBlock), 0, 1,     // id: 105
				105, 30, 2, int(crypto.NoOpBlock), 0, 1,    // id: 135

				// evil takover of the chain
				50, 100, 3, int(crypto.NoOpBlock), 0, 1,    // id: 235
				235, 6, 4, int(crypto.NoOpBlock), 0, 1,},  // id: 241
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(blockReward, txFee, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 50
		mp[Account(strconv.Itoa(3))] = 100
		mp[Account(strconv.Itoa(4))] = 6
		equals(t, mp, bkState.GetAll())
	})
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}