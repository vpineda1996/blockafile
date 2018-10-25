package state

import (
	"../../crypto"
	"../api"
	"log"
	"os"
)

type MinerState interface {
	GetBlock(id string) (*crypto.Block, bool)
	GetFilesystemState(confirmsPerFileCreate int, confirmsPerFileAppend int) (FilesystemState, error)
	GetRoots() []*crypto.Block
	GetAccountState(appendFee int, createFee int, opReward int, noOpReward int) (AccountsState, error)
}

type MinerStateImpl struct {
	Tm      *TreeManager
	clients []*api.MinerClient
}

type Config struct {
	AppendFee             Balance // Note that this is not user-configured. Always exactly 1 coin.
	CreateFee             Balance
	OpReward              Balance
	NoOpReward            Balance
	OpNumberOfZeros       int
	NoOpNumberOfZeros	  int
	Address               string
	ConfirmsPerFileCreate int
	ConfirmsPerFileAppend int
}

var lg = log.New(os.Stdout, "state: ", log.Lmicroseconds|log.Lshortfile)

func (s MinerStateImpl) GetFilesystemState(
	confirmsPerFileCreate int,
	confirmsPerFileAppend int) (FilesystemState, error) {
	return NewFilesystemState(confirmsPerFileCreate, confirmsPerFileAppend, s.Tm.GetLongestChain())
}

func (s MinerStateImpl) GetBlock(id string) (*crypto.Block, bool){
	return s.Tm.GetBlock(id)
}

func (s MinerStateImpl) GetRoots() []*crypto.Block {
	return s.Tm.GetRoots()
}

func (s MinerStateImpl) GetAccountState(
	appendFee int,
	createFee int,
	opReward int,
	noOpReward int) (AccountsState, error) {
	return NewAccountsState(appendFee, createFee, opReward, noOpReward, s.Tm.GetLongestChain())
}

func (s MinerStateImpl) GetRemoteBlock(id string) (*crypto.Block, bool) {
	for _, c := range s.clients {
		nd, ok, err := c.GetBlock(id)
		if err != nil {
			// todo vpineda prob remove that host from the host list
			lg.Printf("error in connection node %v\n", err)
			continue
		}
		if ok && nd.Id() == id {
			return nd, true
		}
	}
	return nil, false
}

func (s MinerStateImpl) GetRemoteRoots() ([]*crypto.Block) {
	blocks := make(map[string]*crypto.Block)
	for _, c := range s.clients {
		arr, err := c.GetRoots()
		if err != nil {
			// todo vpineda prob remove that host from the host list
			lg.Printf("error in connection node %v\n", err)
			continue
		}
		for _, h := range arr {
			blocks[h.Id()] = h
		}
	}

	blockArr := make([]*crypto.Block, len(blocks))
	i := 0
	for _, v := range blocks {
		blockArr[i] = v
		i += 1
	}
	return blockArr
}

func (s MinerStateImpl) OnNewBlock(b *crypto.Block) {
	panic("implement me")
}

func (s MinerStateImpl) OnNewBlockInLongestChain(b *crypto.Block) {
	panic("implement me")
}

func (s MinerStateImpl) AddBlock(b *crypto.Block) {
	lg.Printf("added new block: %x", b.Hash())
	// add it to the tree manager and then broadcast the block
	s.Tm.AddBlock(crypto.BlockElement{
		Block: b,
	})
	// bkst block
	s.broadcastBlock(b)
}

func (s MinerStateImpl) broadcastBlock(b *crypto.Block) {
	go func() {
		for _, c := range s.clients {
			c.SendBlock(b)
		}
	}()
}

func (s MinerStateImpl) AddJob(b *crypto.BlockOp) {
	lg.Printf("added new job: %v", b)
	// todo vpineda add job to miners
	s.broadcastJob(b)
}

func (s MinerStateImpl) broadcastJob(b *crypto.BlockOp) {
	go func() {
		for _, c := range s.clients {
			c.SendJob(b)
		}
	}()
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
		clients: cls,
	}
	var err error
	ms.Tm = NewTreeManager(config, ms, ms)
	if err != nil {
		panic(err)
	}

	err = api.InitMinerServer(config.Address, ms)
	if err != nil {
		panic("cannot init server twice!")
	}
	return ms
}
