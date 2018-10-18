package state

import (
	"../../crypto"
	"../../shared"
	"crypto/md5"
	"log"
	"strconv"
	"testing"
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

func TestSimpleTreeManager(t *testing.T) {
	t.Run("init works", func(t *testing.T) {
		NewTreeManager(Config{
			appendFee: shared.NUM_COINS_PER_FILE_APPEND,
			createFee: 1,
			opReward: 1,
			noOpReward: 1,
			numberOfZeros: numberOfZeros,
		})
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
		})
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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
		})
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
		})
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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
		})
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
		})
		err := buildTreeWithManager(treeDef, tree)

		if err != nil {
			t.Fail()
		}

		fsState, err := NewFilesystemState(tree.GetLongestChain())
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

