package state

import (
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
