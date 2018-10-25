package state

import (
	"../../crypto"
	"../../shared"
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
	height   uint64
	roots    int
	addOrder []int // node where we insert first | the number of nodes we insert | id of miner | tpe | txs | id of creator
}

var genBlockSeed = [md5.Size]byte{10, 2, 1}
var opReward = 1
var noOpReward = 1
var appendFee = shared.NUM_COINS_PER_FILE_APPEND
var createFee = 1

var counter byte = 0

func buildTree(treeDef treeBuilderTest) *MRootTree {

	test := treeDef
	nds := make([]*Node, 0, 100)
	ee := crypto.BlockElement{
		Block: &crypto.Block{
			MinerId:   strconv.Itoa(1),
			Type:      crypto.GenesisBlock,
			PrevBlock: genBlockSeed,
			Records:   []*crypto.BlockOp{},
			Nonce:     12324,
		},
	}
	mtr := NewMRootTree()

	// create a root
	e, _ := mtr.PrependElement(ee, nil)
	nds = append(nds, e)

	for i := 0; i < len(test.addOrder); i += 6 {
		// grab root and start adding n nodes
		root := nds[test.addOrder[i]]
		for j := 0; j < test.addOrder[i+1]; j++ {
			records := make([]*crypto.BlockOp, test.addOrder[i+4])
			for u := 0; u < test.addOrder[i+4]; u++ {
				record := crypto.BlockOp{
					Type:     crypto.CreateFile,
					Filename: "random" + strconv.Itoa(rand.Int()),
					Data:     [crypto.DataBlockSize]byte{counter},
					Creator:  strconv.Itoa(test.addOrder[i+5]),
				}
				records[u] = &record
				counter += 1
			}
			prevBlk := [md5.Size]byte{}
			copy(prevBlk[:], root.Value.(crypto.BlockElement).Block.Hash())
			ee := crypto.BlockElement{
				Block: &crypto.Block{
					MinerId:   strconv.Itoa(test.addOrder[i+2]),
					Type:      crypto.BlockType(test.addOrder[i+3]),
					PrevBlock: prevBlk,
					Records:   records,
					Nonce:     12324,
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
		bkState, _ := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
		equals(t, 0, len(bkState.GetAll()))
	})

	t.Run("simple tree with just the genesis block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height:   1,
			roots:    1,
			addOrder: []int{},
		}
		tree := buildTree(treeDef)

		bkState, _ := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
		equals(t, 0, len(bkState.GetAll()))
	})

	t.Run("simple tree with just genesis and a no-op block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots:  1,
			addOrder: []int{
				0, 1, 1, int(crypto.NoOpBlock), 0, 1},
		}
		tree := buildTree(treeDef)
		bkState, _ := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 1
		equals(t, mp, bkState.GetAll())
	})

	t.Run("simple tree with just genesis, no-op block, op block with 1 tx", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				0, 1, 1, int(crypto.NoOpBlock), 0, 1,
				1, 1, 2, int(crypto.RegularBlock), 1, 1},
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 0
		mp[Account(strconv.Itoa(2))] = 1
		equals(t, mp, bkState.GetAll())
	})

	t.Run("refunds accounts on delete for create", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots:  1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock), 0, 1, 0, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.DeleteFile), 0},
		}
		tree := buildFSTree(treeDef)
		bkState, err := NewAccountsState(3, createFee, opReward, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 102
		equals(t, mp, bkState.GetAll())
	})

	t.Run("refunds accounts on delete for create & append", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots:  1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock), 0, 1, 0, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.AppendFile), 0,
				102, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.DeleteFile), 0},
		}
		tree := buildFSTree(treeDef)
		bkState, err := NewAccountsState(3, createFee, opReward, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 103
		equals(t, mp, bkState.GetAll())
	})

	t.Run("doesn't double refund", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots:  1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock), 0, 1, 0, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.AppendFile), 0,
				102, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.DeleteFile), 0,
				103, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				104, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.AppendFile), 0,
				105, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.DeleteFile), 0},
		}
		tree := buildFSTree(treeDef)
		bkState, err := NewAccountsState(3, createFee, opReward, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 106
		equals(t, mp, bkState.GetAll())
	})
}

func TestComplexBlockChainTree(t *testing.T) {
	t.Run("long branch with multiple accounts", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1},
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
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
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,

				// fake chain
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,

				// longest chain
				100, 1, 5, int(crypto.NoOpBlock), 0, 1, // id: 105
				105, 30, 2, int(crypto.NoOpBlock), 0, 1}, // id: 106
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
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
			roots:  1,
			addOrder: []int{
				// first part chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,

				// fake chain
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,

				// middle chain
				100, 1, 5, int(crypto.NoOpBlock), 0, 1, // id: 105
				105, 30, 2, int(crypto.NoOpBlock), 0, 1, // id: 135

				// evil takover of the chain
				50, 100, 3, int(crypto.NoOpBlock), 0, 1, // id: 235
				235, 6, 4, int(crypto.NoOpBlock), 0, 1}, // id: 241
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
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

func TestAccountConfig(t *testing.T) {
	AddAppendNode := func(tree *MRootTree) {
		hd := tree.GetLongestChain()
		prevBlk := [md5.Size]byte{}
		copy(prevBlk[:], hd.Value.(crypto.BlockElement).Block.Hash())
		records := make([]*crypto.BlockOp, 1)
		record := crypto.BlockOp{
			Type: crypto.AppendFile,
			Filename: filenames[0],
			Data: datum[0],
			Creator: strconv.Itoa(1),
			RecordNumber: uint16(0),
		}
		records[0] = &record
		ee := crypto.BlockElement{
			Block: &crypto.Block{
				MinerId:   strconv.Itoa(1),
				Type:      crypto.RegularBlock,
				PrevBlock: prevBlk,
				Records:   records,
				Nonce:     12324,
			},
		}
		tree.PrependElement(ee, tree.GetLongestChain())
	}

	t.Run("test create fee", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,
				104, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(appendFee, 10, opReward, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 91 // 100 + 1 - 10
		mp[Account(strconv.Itoa(2))] = 1
		mp[Account(strconv.Itoa(3))] = 3
		equals(t, mp, bkState.GetAll())
	})

	t.Run("test append fee", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,
				104, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := buildTree(treeDef)

		// Add append node
		AddAppendNode(tree)

		// Strictly speaking the AppendFee should always == 1, but for testing purposes we set it to something
		// larger here
		bkState, err := NewAccountsState(10, createFee, opReward, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 91 // 100 + 1 - 10
		mp[Account(strconv.Itoa(2))] = 1
		mp[Account(strconv.Itoa(3))] = 3
		equals(t, mp, bkState.GetAll())
	})

	t.Run("test op reward", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,
				104, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := buildTree(treeDef)
		// weooo that's a lot of money!
		bkState, err := NewAccountsState(appendFee, createFee, 1000, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 1099 // 100 + 1000 - 1
		mp[Account(strconv.Itoa(2))] = 1
		mp[Account(strconv.Itoa(3))] = 3
		equals(t, mp, bkState.GetAll())
	})

	t.Run("test noop reward", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,
				104, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := buildTree(treeDef)
		// if only making money was this easy
		bkState, err := NewAccountsState(appendFee, createFee, opReward, 100, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 10000 // 100 * 100 + 1 - 1
		mp[Account(strconv.Itoa(2))] = 100
		mp[Account(strconv.Itoa(3))] = 300
		equals(t, mp, bkState.GetAll())
	})

	t.Run("test reward and fee in the same tree", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,
				104, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(appendFee, 10, opReward, 10, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 991 // 100 * 10 - 10 + 1
		mp[Account(strconv.Itoa(2))] = 10
		mp[Account(strconv.Itoa(3))] = 30
		equals(t, mp, bkState.GetAll())
	})

	t.Run("test reward and fee in the same block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 2,
			roots:  1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 1, 2, int(crypto.NoOpBlock), 0, 1,
				101, 3, 3, int(crypto.NoOpBlock), 0, 1,
				104, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := buildTree(treeDef)
		bkState, err := NewAccountsState(appendFee, 10, 5, noOpReward, tree.GetLongestChain())
		if err != nil {
			t.Fatal(err)
			t.Fail()
		}
		mp := make(map[Account]Balance)
		mp[Account(strconv.Itoa(1))] = 95 // 100 + 5 - 10
		mp[Account(strconv.Itoa(2))] = 1
		mp[Account(strconv.Itoa(3))] = 3
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
