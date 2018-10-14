package state

import (
	"../../shared/datastruct"
	"log"
	"os"
)

// TODO implement state that holds the BlockChainStatus and FileSystemState

type State struct {
	tm *TreeManager
}

var lg = log.New(os.Stdout, "state: ", log.Lmicroseconds|log.Lshortfile)

func GetFileSystemState() {
	// TODO get a node from tm and call new in the fs state
}

func GetBlockChainState()  {
	// TODO get a node from tm and call new in the blockchain state
}


func transverseChain(root *datastruct.Node) []*datastruct.Node {
	res := make([]*datastruct.Node, root.Height + 1)
	// create list
	for nd, i := root, 0; nd != nil; nd, i = nd.Next(), i + 1 {
		res[i] = nd
	}
	// reverse
	for l, r := 0, len(res) - 1; l < r; l, r = l + 1, r - 1 {
		res[l], res[r] = res[r], res[l]
	}

	return res
}