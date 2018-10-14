package state

import (
	"../../crypto"
	"crypto/md5"
	"strconv"
	"testing"
)

import (
	. "../../shared/datastruct"
)


// add order : node where we insert first | the number of nodes we insert
// 										       | id of miner | tpe
// 											   | txs | id of creator | idx of dataArr |
//											   | idx for filenames | fsop

var filenames = []string{"a", "b", "c", "d"}
var datum = [][crypto.DataBlockSize]byte{
	{1, 2, 3, 4},
	{5, 6, 7, 8},
	{9, 10,11,12},
	{13,14,15,16},
}

func buildFSTree(treeDef treeBuilderTest) *MRootTree {

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

	for i := 0; i < len(test.addOrder); i+= 9 {
		// grab root and start adding n nodes
		root := nds[test.addOrder[i]]
		for j := 0; j < test.addOrder[i+1]; j++ {
			records := make([]*crypto.BlockOp, test.addOrder[i+4])
			for u := 0; u < test.addOrder[i+4]; u++ {
				record := crypto.BlockOp{
					Type: crypto.BlockOpType(test.addOrder[i+8]),
					Filename: filenames[test.addOrder[i+7]],
					Data: datum[test.addOrder[i+6]],
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

func TestSimpleFilesystemTree(t *testing.T) {
	t.Run("returns empty state on empty tree", func(t *testing.T) {
		tree := NewMRootTree()
		fsState, _ := NewFilesystemState(tree.GetLongestChain())
		equals(t, 0, len(fsState.GetAll()))
	})

	t.Run("simple tree with just the genesis block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{},
		}
		tree := buildTree(treeDef)

		fsState, _ := NewFilesystemState(tree.GetLongestChain())
		equals(t, 0, len(fsState.GetAll()))
	})

	t.Run("simple tree with just genesis and a no-op block", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 1, 1, int(crypto.NoOpBlock), 0, 1},
		}
		tree := buildTree(treeDef)
		fsState, _ := NewFilesystemState(tree.GetLongestChain())
		mp := make(map[Filename]*FileInfo)
		equals(t, mp, fsState.GetAll())
	})

	t.Run("simple tree with just genesis, a no-op block, and a record", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock), 0, 1,
				100, 5, 1, int(crypto.RegularBlock), 1, 1},
		}
		tree := buildTree(treeDef)
		fsState, _ := NewFilesystemState(tree.GetLongestChain())
		equals(t, 5, len(fsState.GetAll()))
	})

	t.Run("simple tree with just genesis, a no-op block, and a record", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile)},
		}
		tree := buildFSTree(treeDef)
		fsState, err := NewFilesystemState(tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 1, len(fs))
		equals(t, "1", fs["a"].Creator)
	})

	t.Run("simple tree with just genesis, a no-op block, a record and append", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile),
				101, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.AppendFile)},
		}
		tree := buildFSTree(treeDef)
		fsState, err := NewFilesystemState(tree.GetLongestChain())
		if err != nil {
			panic(err)
		}
		fs := fsState.GetAll()
		equals(t, 1, len(fs))
		equals(t, "1", fs["a"].Creator)
		equals(t, datum[0][:], []byte(fs["a"].Data))
	})

	t.Run("fails if we try to create more than two files", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				100, 2, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile)},
		}
		tree := buildFSTree(treeDef)
		_, err := NewFilesystemState(tree.GetLongestChain())
		if err == nil {
			t.Fail()
		}
	})
}

func TestComplexFilesystemTree(t *testing.T) {
	t.Run("long branch with multiple files with no append", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// true chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile),
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile),
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile),},
		}
		tree := buildFSTree(treeDef)
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
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile),
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile),
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile),
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile),
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				120, 1, 2, int(crypto.RegularBlock), 1, 1, 1, 2, int(crypto.AppendFile),},
		}
		tree := buildFSTree(treeDef)
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
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile),
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile),
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile),
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile),
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				120, 1, 2, int(crypto.RegularBlock), 1, 2, 1, 2, int(crypto.AppendFile),},
		}
		tree := buildFSTree(treeDef)
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

	t.Run("multiple chains, longest chain keeps state of the fs", func(t *testing.T) {
		treeDef := treeBuilderTest{
			height: 1,
			roots: 1,
			addOrder: []int{
				// first chain
				0, 100, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				100, 1, 1, int(crypto.RegularBlock), 1, 1, 0, 0, int(crypto.CreateFile),
				101, 5, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				106, 1, 1, int(crypto.RegularBlock), 1, 2, 0, 1, int(crypto.CreateFile),
				107, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.CreateFile), // id 108

				// divergence into another root
				108, 2, 2, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				110, 1, 2, int(crypto.RegularBlock), 1, 1, 0, 2, int(crypto.AppendFile),
				111, 9, 1, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,
				120, 1, 2, int(crypto.RegularBlock), 1, 2, 1, 2, int(crypto.AppendFile),

				// appends happen on that branch but somebody decided to be evil
				108, 79, 3, int(crypto.NoOpBlock),    0, 1, 0, 0, 0,                   // id 200
				200, 1,  3, int(crypto.RegularBlock), 1, 3, 3, 2, int(crypto.AppendFile),
				},
		}
		tree := buildFSTree(treeDef)
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
