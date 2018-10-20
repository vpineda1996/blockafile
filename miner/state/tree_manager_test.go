package state

import (
	"../../crypto"
	"../../shared"
	"crypto/md5"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// add order : node where we insert first | the number of nodes we insert
// 										       | id of miner | tpe
// 											   | txs | id of creator | idx of dataArr |
//											   | idx for filenames | fsop | append no

func buildTreeWithManager(treeDef treeBuilderTest, tm *TreeManager) error {
	test := treeDef
	ndIds := make([][md5.Size]byte, 0, 100)
	ee := crypto.BlockElement{
		Block: &crypto.Block {
			MinerId: strconv.Itoa(1),
			Type: crypto.GenesisBlock,
			PrevBlock: genBlockSeed,
			Records: []*crypto.BlockOp{},
			Nonce: 12324,
		},
	}
	// add genesis block
	tm.AddBlock(ee)
	buf := [md5.Size]byte{}
	copy(buf[:], ee.Block.Hash())
	ndIds = append(ndIds, buf)

	for i := 0; i < len(test.addOrder); i+= 10 {
		// grab root and start adding n nodes
		rootId := ndIds[test.addOrder[i]]
		for j := 0; j < test.addOrder[i+1]; j++ {
			records := make([]*crypto.BlockOp, test.addOrder[i+4])
			for u := 0; u < test.addOrder[i+4]; u++ {
				record := crypto.BlockOp{
					Type: crypto.BlockOpType(test.addOrder[i+8]),
					Filename: filenames[test.addOrder[i+7]],
					Data: datum[test.addOrder[i+6]],
					Creator: strconv.Itoa(test.addOrder[i+5]),
					RecordNumber: uint32(test.addOrder[i+9]) + uint32(u),
				}
				records[u] = &record
				counter += 1
			}
			ee := crypto.BlockElement{
				Block: &crypto.Block {
					MinerId: strconv.Itoa(test.addOrder[i+2]),
					Type: crypto.BlockType(test.addOrder[i+3]),
					PrevBlock: rootId,
					Records: records,
					Nonce: 12324,
				},
			}
			ee.Block.FindNonce(numberOfZeros)
			var err error
			err = tm.AddBlock(ee)
			if err != nil {
				return err
			}
			buf := [md5.Size]byte{}
			copy(buf[:], ee.Block.Hash())
			ndIds = append(ndIds, buf)
			rootId = buf
		}
	}
	return nil
}

const numberOfZeros = 8

type fakeNodeRetrievier struct {

}

func (fakeNodeRetrievier) OnNewBlockInTree(b *crypto.Block) {
}

func (fakeNodeRetrievier) OnNewBlockInLongestChain(b *crypto.Block) {
}

func (fakeNodeRetrievier) GetRemoteBlock(id string) (*crypto.Block, bool) {
	panic("implement me")
}

func (fakeNodeRetrievier) GetRemoteRoots() ([]*crypto.Block) {
	return []*crypto.Block{}
}

var fkNodeRetriv = fakeNodeRetrievier{}

func TestSimpleTreeManager(t *testing.T) {
	t.Run("init works", func(t *testing.T) {
		NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
	})

	t.Run("simple tree with just the genesis block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		bkState, _ := NewAccountsState(appendFee, createFee, opReward, noOpReward, tree.GetLongestChain())
		equals(t, 0, len(bkState.GetAll()))
	})

	t.Run("simple tree with just genesis, a no-op block, and a record", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 1, len(fs))
		equals(t, "1", fs["a"].Creator)
	})

	t.Run("simple tree with just genesis, a record", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 1, len(fs))
		equals(t, "1", fs["a"].Creator)
	})

	t.Run("fails if account doesn't have money", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 0, int(crypto.CreateFile), 0},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err == nil {
			t.Fail()
		}
	})

	t.Run("fails if account doesnt have money for all tnx described in block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 2, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				2, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				3, 1, 2, int(crypto.RegularBlock), 2, 1, 0, 0, int(crypto.AppendFile), 0},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err == nil {
			t.Fail()
		}
	})

	t.Run("simple tree with just genesis, a no-op block, a record and append", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.AppendFile), 0},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			log.Println(err)
			t.Fail()
		}
		fs := fsState.GetAll()
		equals(t, 1, len(fs))
		equals(t, "1", fs["a"].Creator)
		equals(t, datum[0][:], []byte(fs["a"].Data))
	})

	t.Run("simple tree with just genesis, a no-op block, a record and append, multiple recs in block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 1, int(crypto.RegularBlock), 5, 1, 0, 0, int(crypto.AppendFile), 0},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			log.Println(err)
			t.Fail()
		}
		fs := fsState.GetAll()
		equals(t, 1, len(fs))
		equals(t, "1", fs["a"].Creator)
		equals(t, datum[0][:], []byte(fs["a"].Data)[:crypto.DataBlockSize])
		equals(t, uint32(5), fs["a"].NumberOfRecords)
	})
}

func TestValidTnxTreeManager(t *testing.T) {
	t.Run("long branch with multiple files with no append", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile), 0,
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile), 0,},
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 3, len(fs))

		equals(t, "1", fs["a"].Creator)
		equals(t, "2", fs["b"].Creator)
		equals(t, "1", fs["c"].Creator)
	})

	t.Run("long branch with multiple files with append, single user", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile), 0,
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile), 0,
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile), 0,
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				120, 1, 2, int(crypto.RegularBlock), 1, 1, 1, 2, int(crypto.AppendFile), 1,},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 3, len(fs))

		equals(t, "1", fs["a"].Creator)

		equals(t, "2", fs["b"].Creator)

		equals(t, "1", fs["c"].Creator)

		equals(t, datum[0][:], []byte(fs["c"].Data)[:crypto.DataBlockSize])
		equals(t, datum[1][:], []byte(fs["c"].Data)[crypto.DataBlockSize:])
	})

	t.Run("long branch with multiple files with append, multi user append", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile), 0,
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile), 0,
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile), 0,
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				120, 1, 2, int(crypto.RegularBlock), 1, 2, 1, 2, int(crypto.AppendFile), 1,},
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 3, len(fs))

		equals(t, "1", fs["a"].Creator)

		equals(t, "2", fs["b"].Creator)

		equals(t, "1", fs["c"].Creator)

		equals(t, datum[0][:], []byte(fs["c"].Data)[:crypto.DataBlockSize])
		equals(t, datum[1][:], []byte(fs["c"].Data)[crypto.DataBlockSize:])
	})

	t.Run("fails to create a tree with conflicting appends", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile), 0,
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile), 0,
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile), 0,
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0,
				120, 1, 2, int(crypto.RegularBlock), 1, 2, 1, 2, int(crypto.AppendFile), 0,},
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err == nil {
			t.Fail()
		}
	})


	t.Run("multiple chains, longest chain keeps state of the fs", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// first chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile),  0,
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile),  0,
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile),  0, // id 108

				// divergence into another root
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile),  0,
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				120, 1, 2, int(crypto.RegularBlock), 1, 2, 1, 2, int(crypto.AppendFile),  1,

				// appends happen on that branch but somebody decided to be evil
				108, 79, 3, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0, // id 200
				200, 1,  3, int(crypto.RegularBlock), 1, 3, 3, 2, int(crypto.AppendFile), 0,
			},
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(0, 0, tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 3, len(fs))

		equals(t, "1", fs["a"].Creator)

		equals(t, "2", fs["b"].Creator)

		equals(t, "1", fs["c"].Creator)
		equals(t, datum[3][:], []byte(fs["c"].Data)[:])
	})

}

func TestValidAccountState(t *testing.T) {
	t.Run("test reward and fee in the same block, fails", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				100, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 2, int(crypto.RegularBlock), 2, 2, 0, 0, int(crypto.AppendFile), 0},
		}
		// Strictly speaking the appendFee should always == 1, but for testing purposes we set it to something
		// larger here
		tree := NewTreeManager(Config{
			appendFee: 1000,
			createFee: 1,
			opReward: 500,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err == nil {
			t.Fail()
		}
	})

	t.Run("test reward and fee in the same block, fine", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				100, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 2, int(crypto.RegularBlock), 2, 2, 0, 0, int(crypto.AppendFile), 0},
		}
		// Strictly speaking the appendFee should always == 1, but for testing purposes we set it to something
		// larger here
		tree := NewTreeManager(Config{
			appendFee: 1000,
			createFee: 1,
			opReward: 1000,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}
	})

	t.Run("fails if append is too costly", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				100, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 2, int(crypto.RegularBlock), 2, 2, 0, 0, int(crypto.AppendFile), 0},
		}
		// Strictly speaking the appendFee should always == 1, but for testing purposes we set it to something
		// larger here
		tree := NewTreeManager(Config{
			appendFee: 1000,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err == nil {
			t.Fail()
		}
	})

	t.Run("fails if create is too costly", func(t *testing.T){
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0, 0,
				100, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile), 0,
				101, 1, 2, int(crypto.RegularBlock), 2, 2, 0, 0, int(crypto.AppendFile), 0},
		}
		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1000,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, fkNodeRetriv)
		err := buildTreeWithManager(treeDef, tree)

		if err == nil {
			t.Fail()
		}
	})
}

type tNodeRetriever struct {
	counterRB *int
	counterRR *int
	block *crypto.Block
	block2 *crypto.Block
}

func (t tNodeRetriever) GetRemoteBlock(id string) (*crypto.Block, bool) {
	*t.counterRB += 1
	if fmt.Sprintf("%x", t.block.Hash()) == id {
		return t.block, true
	}
	if t.block2 != nil && fmt.Sprintf("%x", t.block2.Hash()) == id  {
		return t.block2, true
	}
	return nil, false
}

func (t tNodeRetriever) GetRemoteRoots() ([]*crypto.Block) {
	*t.counterRR += 1
	ee := crypto.BlockElement{
		Block: &crypto.Block {
			MinerId: strconv.Itoa(1),
			Type: crypto.GenesisBlock,
			PrevBlock: genBlockSeed,
			Records: []*crypto.BlockOp{},
			Nonce: 12324,
		},
	}
	return []*crypto.Block{ee.Block}
}

var cGenBlockSeed = [md5.Size]byte{10, 2,1, 5}

func TestBlockRetrieval(t *testing.T) {
	t.Run("it gets the parent block", func(t *testing.T) {
		parent := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: strconv.Itoa(1),
				Type: crypto.NoOpBlock,
				PrevBlock: genBlockSeed,
				Records: []*crypto.BlockOp{},
				Nonce: 12324,
			},
		}

		parent.Block.FindNonce(numberOfZeros)
		parentHs := [md5.Size]byte{}
		copy(parentHs[:], parent.Block.Hash())

		head := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: "1",
				Type: crypto.RegularBlock,
				PrevBlock: parentHs,
				Records: []*crypto.BlockOp{{
					Type: crypto.CreateFile,
					RecordNumber: 0,
					Filename: "potato",
					Creator: "1",
					Data: [512]byte{},
				}},
				Nonce: 12324,
			},
		}
		head.Block.FindNonce(numberOfZeros)

		var tNodeRetrivStruct = tNodeRetriever{
			block: parent.Block,
			counterRB: new(int),
			counterRR: new(int),
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, tNodeRetrivStruct, fkNodeRetriv)
		tree.StartThreads()
		time.Sleep(time.Millisecond * 100)

		err := tree.AddBlock(head)
		ok(t, err)

		time.Sleep(time.Millisecond * 100)

		equals(t, 1, *tNodeRetrivStruct.counterRB)

		fsState, err := NewFilesystemState(0, 0 , tree.GetLongestChain())
		ok(t, err)

		fs := fsState.GetAll()
		equals(t, 1, len(fs))
		equals(t, "1", fs["potato"].Creator)
	})

	t.Run("discards block if parent is garbage", func(t *testing.T) {
		parent := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: strconv.Itoa(1),
				Type: crypto.NoOpBlock,
				PrevBlock: genBlockSeed,
				Records: []*crypto.BlockOp{},
				Nonce: 12324,
			},
		}

		parentHs := [md5.Size]byte{}
		copy(parentHs[:], parent.Block.Hash())

		head := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: "1",
				Type: crypto.RegularBlock,
				PrevBlock: parentHs,
				Records: []*crypto.BlockOp{{
					Type: crypto.CreateFile,
					RecordNumber: 0,
					Filename: "potato",
					Creator: "1",
					Data: [512]byte{},
				}},
				Nonce: 12324,
			},
		}
		head.Block.FindNonce(numberOfZeros)

		var tNodeRetrivStruct = tNodeRetriever{
			block: parent.Block,
			counterRB: new(int),
			counterRR: new(int),
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, tNodeRetrivStruct, fkNodeRetriv)
		tree.StartThreads()
		time.Sleep(time.Millisecond * 100)

		err := tree.AddBlock(head)
		ok(t, err)

		time.Sleep(time.Millisecond * 100)

		equals(t, 1, *tNodeRetrivStruct.counterRB)

		fsState, err := NewFilesystemState(0, 0 , tree.GetLongestChain())
		ok(t, err)

		fs := fsState.GetAll()
		equals(t, 0, len(fs))
	})

	t.Run("corrupt seed on node", func(t *testing.T) {

		parent := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: strconv.Itoa(1),
				Type: crypto.NoOpBlock,
				PrevBlock: cGenBlockSeed,
				Records: []*crypto.BlockOp{},
				Nonce: 12324,
			},
		}
		parent.Block.FindNonce(numberOfZeros)
		parentHs := [md5.Size]byte{}
		copy(parentHs[:], parent.Block.Hash())

		head := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: "1",
				Type: crypto.RegularBlock,
				PrevBlock: parentHs,
				Records: []*crypto.BlockOp{{
					Type: crypto.CreateFile,
					RecordNumber: 0,
					Filename: "potato",
					Creator: "1",
					Data: [512]byte{},
				}},
				Nonce: 12324,
			},
		}
		head.Block.FindNonce(numberOfZeros)

		var tNodeRetrivStruct = tNodeRetriever{
			block: parent.Block,
			counterRB: new(int),
			counterRR: new(int),
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, tNodeRetrivStruct, fkNodeRetriv)
		tree.StartThreads()
		time.Sleep(time.Millisecond * 100)

		err := tree.AddBlock(head)
		ok(t, err)

		time.Sleep(time.Millisecond * 100)

		fsState, err := NewFilesystemState(0, 0 , tree.GetLongestChain())
		ok(t, err)

		fs := fsState.GetAll()
		equals(t, 0, len(fs))
	})

	t.Run("long chain works", func(t *testing.T) {
		parent := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: strconv.Itoa(1),
				Type: crypto.NoOpBlock,
				PrevBlock: genBlockSeed,
				Records: []*crypto.BlockOp{},
				Nonce: 12324,
			},
		}

		parent.Block.FindNonce(numberOfZeros)
		parentHs := [md5.Size]byte{}
		copy(parentHs[:], parent.Block.Hash())

		head := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: "1",
				Type: crypto.RegularBlock,
				PrevBlock: parentHs,
				Records: []*crypto.BlockOp{{
					Type: crypto.CreateFile,
					RecordNumber: 0,
					Filename: "potato",
					Creator: "1",
					Data: [512]byte{},
				}},
				Nonce: 12324,
			},
		}

		head.Block.FindNonce(numberOfZeros)
		head2Parent := [md5.Size]byte{}
		copy(head2Parent[:], head.Block.Hash())

		head2 := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: "1",
				Type: crypto.RegularBlock,
				PrevBlock: head2Parent,
				Records: []*crypto.BlockOp{{
					Type: crypto.CreateFile,
					RecordNumber: 0,
					Filename: "potato2",
					Creator: "1",
					Data: [512]byte{},
				}},
				Nonce: 12324,
			},
		}
		head2.Block.FindNonce(numberOfZeros)

		var tNodeRetrivStruct = tNodeRetriever{
			block: parent.Block,
			block2: head.Block,
			counterRB: new(int),
			counterRR: new(int),
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, tNodeRetrivStruct, fkNodeRetriv)
		tree.StartThreads()
		time.Sleep(time.Millisecond * 100)

		err := tree.AddBlock(head2)
		ok(t, err)

		time.Sleep(time.Millisecond * 100)

		equals(t, 2, *tNodeRetrivStruct.counterRB)

		fsState, err := NewFilesystemState(0, 0 , tree.GetLongestChain())
		ok(t, err)

		fs := fsState.GetAll()
		equals(t, 2, len(fs))
		equals(t, "1", fs["potato"].Creator)
	})

	t.Run("fails gracefully with long chain", func(t *testing.T) {
		parent := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: strconv.Itoa(1),
				Type: crypto.NoOpBlock,
				PrevBlock: cGenBlockSeed,
				Records: []*crypto.BlockOp{},
				Nonce: 12324,
			},
		}

		parent.Block.FindNonce(numberOfZeros)
		parentHs := [md5.Size]byte{}
		copy(parentHs[:], parent.Block.Hash())

		head := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: "1",
				Type: crypto.RegularBlock,
				PrevBlock: parentHs,
				Records: []*crypto.BlockOp{{
					Type: crypto.CreateFile,
					RecordNumber: 0,
					Filename: "potato",
					Creator: "1",
					Data: [512]byte{},
				}},
				Nonce: 12324,
			},
		}

		head.Block.FindNonce(numberOfZeros)
		head2Parent := [md5.Size]byte{}
		copy(head2Parent[:], head.Block.Hash())

		head2 := crypto.BlockElement{
			Block: &crypto.Block {
				MinerId: "1",
				Type: crypto.RegularBlock,
				PrevBlock: head2Parent,
				Records: []*crypto.BlockOp{{
					Type: crypto.CreateFile,
					RecordNumber: 0,
					Filename: "potato2",
					Creator: "1",
					Data: [512]byte{},
				}},
				Nonce: 12324,
			},
		}
		head2.Block.FindNonce(numberOfZeros)

		var tNodeRetrivStruct = tNodeRetriever{
			block: parent.Block,
			block2: head.Block,
			counterRB: new(int),
			counterRR: new(int),
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, tNodeRetrivStruct, fkNodeRetriv)
		tree.StartThreads()
		time.Sleep(time.Millisecond * 100)

		err := tree.AddBlock(head2)
		ok(t, err)

		time.Sleep(time.Millisecond * 100)

		equals(t, 3, *tNodeRetrivStruct.counterRB)

		fsState, err := NewFilesystemState(0, 0 , tree.GetLongestChain())
		ok(t, err)

		fs := fsState.GetAll()
		equals(t, 0, len(fs))
	})
}

type obl struct {
	newb *int
	newll *int
}

func (o obl) OnNewBlockInTree(b *crypto.Block) {
	*o.newb += 1
}

func (o obl) OnNewBlockInLongestChain(b *crypto.Block) {
	*o.newll += 1
}

func TestOnBlockListeners(t *testing.T) {
	t.Run("calls on new block when adding genesis block", func(t *testing.T) {
		ob := obl{
			newll: new(int),
			newb: new(int),
		}

		var tNodeRetrivStruct = tNodeRetriever{
			block: nil,
			block2: nil,
			counterRB: new(int),
			counterRR: new(int),
		}

		tm := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, tNodeRetrivStruct, ob)
		tm.StartThreads()
		time.Sleep(time.Millisecond * 100)

		// retrieve node implicitly via getRoots thread

		equals(t, 1, *ob.newb)
		equals(t, 1, *ob.newll)
	})

	t.Run("multiple chains, only calls new root when adding to longest chain", func(t *testing.T) {
		ob := obl{
			newll: new(int),
			newb: new(int),
		}

		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// first chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile),  0,
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile),  0,
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile),  0, // id 108

				// divergence into another root
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile),  0,
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                       0,
				120, 1, 2, int(crypto.RegularBlock), 1, 2, 1, 2, int(crypto.AppendFile),  1,

				// appends happen on that branch but somebody decided to be evil
				108, 79, 3, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                      0, // id 200
				200, 1,  3, int(crypto.RegularBlock), 1, 3, 3, 2, int(crypto.AppendFile), 0,
			},
		}

		tree := NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		}, fkNodeRetriv, ob)
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}
		equals(t, 202, *ob.newb)
		equals(t, 202 - 12, *ob.newll)
	})
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d: unexpected error: %str\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}
