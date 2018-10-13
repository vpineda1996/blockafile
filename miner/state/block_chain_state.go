package state

import (
	"../../shared/datastruct"
)

type Account string
type Balance int

type BlockChainState struct {
	accounts map[Account]Balance
}

func NewBlockChainState(nd *datastruct.Node) BlockChainState {
	return BlockChainState{}
}

// TODO Given a node (ie the top of the largest chain), it will generate a state
// TODO with all of the accounts