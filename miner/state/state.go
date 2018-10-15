package state

import (
	"../../crypto"
	"../api"
	"log"
	"os"
)

type State struct {
	tm *TreeManager
	clients []*api.MinerClient
}

type Config struct {
	txFee Balance
	reward Balance
	numberOfZeros int
}

var lg = log.New(os.Stdout, "state: ", log.Lmicroseconds|log.Lshortfile)

func (s *State) GetFilesystemState() (FilesystemState, error) {
	return NewFilesystemState(s.tm.GetLongestChain())
}

func (s *State) GetNode(id string) (*crypto.Block, bool){
	return s.tm.GetBlock(id)
}

func (s *State) GetRoots() []*crypto.Block {
	return s.tm.GetRoots()
}

func (s *State) GetAccountState(txFee int, reward int) (AccountsState, error) {
	return NewAccountsState(reward, txFee, s.tm.GetLongestChain())
}
