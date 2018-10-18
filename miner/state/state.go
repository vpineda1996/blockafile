package state

import (
	"../../crypto"
	"../api"
	"log"
	"os"
)

type MinerState interface {
	GetFilesystemState() (FilesystemState, error)
	GetNode(id string) (*crypto.Block, bool)
	GetRoots() []*crypto.Block
	GetAccountState(appendFee int, createFee int, opReward int, noOpReward int) (AccountsState, error)
}

type MinerStateImpl struct {
	tm *TreeManager
	clients []*api.MinerClient
}

type Config struct {
	appendFee Balance // Note that this is not user-configured. Always exactly 1 coin.
	createFee Balance
	opReward Balance
	noOpReward Balance
	numberOfZeros int
	address string
}

var lg = log.New(os.Stdout, "state: ", log.Lmicroseconds|log.Lshortfile)

func (s MinerStateImpl) GetFilesystemState() (FilesystemState, error) {
	return NewFilesystemState(s.tm.GetLongestChain())
}

func (s MinerStateImpl) GetNode(id string) (*crypto.Block, bool){
	return s.tm.GetBlock(id)
}

func (s MinerStateImpl) GetRoots() []*crypto.Block {
	return s.tm.GetRoots()
}

func (s MinerStateImpl) GetAccountState(
	appendFee int,
	createFee int,
	opReward int,
	noOpReward int) (AccountsState, error) {
	return NewAccountsState(appendFee, createFee, opReward, noOpReward, s.tm.GetLongestChain())
}

func (s MinerStateImpl) AddBlock(b *crypto.Block) {
	lg.Printf("added new block: %x", b.Hash())
	s.tm.AddBlock(crypto.BlockElement{
		Block: b,
	})
}

func (s MinerStateImpl) AddJob(b *crypto.BlockOp) {
	lg.Printf("added new job: %v", b)
	// todo vpineda add job to miners
}

func NewMinerState(config Config, connectedMiningNodes []string) MinerState {
	cls := make([]*api.MinerClient, 0, len(connectedMiningNodes))
	for _, c := range connectedMiningNodes {
		conn, err := api.NewMinerCliet(c)
		if err == nil {
			cls = append(cls, &conn)
		}
	}
	ms := MinerStateImpl{
		tm: NewTreeManager(config),
		clients: cls,
	}

	err := api.InitMinerServer(config.address, ms)
	if err != nil {
		panic("cannot init server twice!")
	}
	return ms
}
