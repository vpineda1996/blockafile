package state

import (
	"../../crypto"
	"../../shared"
	"../api"
	. "../block_calculators"
	"crypto/md5"
	"fmt"
	"github.com/DistributedClocks/GoVector/govec"
	"log"
	"os"
	"time"
)

type MinerState struct {
	// dont ask of the double ptr, its cancer but there is no other way
	logger  *govec.GoLog
	tm      **TreeManager
	clients *[]*api.MinerClient
	bc      **BlockCalculator
	minerId string
}

type Config struct {
	AppendFee             Balance // Note that this is not user-configured. Always exactly 1 coin.
	CreateFee             Balance
	OpReward              Balance
	NoOpReward            Balance
	NumberOfZeros         int
	Address               string
	ConfirmsPerFileCreate int
	ConfirmsPerFileAppend int
	OpPerBlock            int
	MinerId               string
	GenesisBlockHash      [md5.Size]byte
	GenOpBlockTimeout     uint8
}

var lg = log.New(os.Stdout, "state: ", log.Lmicroseconds|log.Lshortfile)
var INFO = govec.GoLogOptions{Priority: govec.INFO}
var ERR = govec.GoLogOptions{Priority: govec.ERROR}
var WARN = govec.GoLogOptions{Priority: govec.WARNING}

func (s MinerState) GetFilesystemState(
	confirmsPerFileCreate int,
	confirmsPerFileAppend int) (FilesystemState, error) {
	return NewFilesystemState(confirmsPerFileCreate, confirmsPerFileAppend, (*s.tm).GetLongestChain())
}

func (s MinerState) GetBlock(id string) (*crypto.Block, bool) {
	return (*s.tm).GetBlock(id)
}

func (s MinerState) GetRoots() []*crypto.Block {
	return (*s.tm).GetRoots()
}

func (s MinerState) GetAccountState(
	appendFee int,
	createFee int,
	opReward int,
	noOpReward int) (AccountsState, error) {
	return NewAccountsState(appendFee, createFee, opReward, noOpReward, (*s.tm).GetLongestChain())
}

func (s MinerState) GetRemoteBlock(id string) (*crypto.Block, bool) {
	for _, c := range *s.clients {
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

func (s MinerState) GetRemoteRoots() []*crypto.Block {
	blocks := make(map[string]*crypto.Block)
	for _, c := range *s.clients {
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

// call from the tree when a block was confirmed and added to the tree
func (s MinerState) OnNewBlockInTree(b *crypto.Block) {
	// notify calculators
	s.logger.LogLocalEvent(fmt.Sprintf(" Block %v added to tree", b.Id()), INFO)
	(*s.bc).RemoveJobsFromBlock(b)
}

func (s MinerState) OnNewBlockInLongestChain(b *crypto.Block) {
	// todo notify to any listener
	s.logger.LogLocalEvent(fmt.Sprintf(" New head on longest chain: %v", b.Id()), INFO)
}

func (s MinerState) AddBlock(b *crypto.Block) {
	// add it to the tree manager and then broadcast the block
	if !(*s.tm).Exists(b) {
		err := (*s.tm).AddBlock(crypto.BlockElement{
			Block: b,
		})
		if err != nil {
			return
		}
		// bkst block
		s.broadcastBlock(b)
	} else {
		s.logger.LogLocalEvent(fmt.Sprintf(" Recieved block %v but I have it", b.Id()), WARN)
	}
}

func (s MinerState) broadcastBlock(b *crypto.Block) {
	go func() {
		for _, c := range *s.clients {
			c.SendBlock(b)
		}
	}()
}

func (s MinerState) AddJob(b crypto.BlockOp) {
	lg.Printf("Added new job: %v", b.Filename)
	if (*s.bc).JobExists(&b) < 0 {
		s.logger.LogLocalEvent(fmt.Sprintf(" Enqueuing job for file %v and record %v for miner to work on", b.Filename, b.RecordNumber), INFO)
		(*s.bc).AddJob(&b)
		s.broadcastJob(&b)
	} else {
		s.logger.LogLocalEvent(fmt.Sprintf(" Recieved job for file %v but I have it", b.Filename), WARN)
	}

}

func (s MinerState) broadcastJob(b *crypto.BlockOp) {
	go func() {
		for _, c := range *s.clients {
			c.SendJob(b)
		}
	}()
}

func (s MinerState) GetHighestRoot() *crypto.Block {
	return (*s.tm).GetHighestRoot()
}

func (s MinerState) GetMinerId() string {
	return s.minerId
}

func (s MinerState) ValidateJobSet(bOps []*crypto.BlockOp) []*crypto.BlockOp {
	return (*s.tm).ValidateJobSet(bOps)
}

func NewMinerState(config Config, connectedMiningNodes []string) MinerState {
	logger := govec.InitGoVector(config.MinerId, shared.LOGFILE, shared.GoVecOpts)
	cls := make([]*api.MinerClient, 0, len(connectedMiningNodes))
	for _, c := range connectedMiningNodes {
		conn, err := api.NewMinerClient(c, logger)
		if err == nil {
			cls = append(cls, &conn)
		}
	}
	var treePtr *TreeManager
	var blockCalcPtr *BlockCalculator
	ms := MinerState{
		clients: &cls,
		minerId: config.MinerId,
		tm:      &treePtr,
		bc:      &blockCalcPtr,
		logger:  logger,
	}
	treePtr = NewTreeManager(config, ms, ms)
	blockCalcPtr = NewBlockCalculator(ms, config.NumberOfZeros, config.OpPerBlock, time.Duration(config.GenOpBlockTimeout))

	// add genesis block
	err := (*ms.tm).AddBlock(crypto.BlockElement{
		Block: &crypto.Block{
			Records:   []*crypto.BlockOp{},
			Type:      crypto.GenesisBlock,
			PrevBlock: config.GenesisBlockHash,
			Nonce:     0,
			MinerId:   "",
		},
	})

	if err != nil {
		panic("cannot add genesis block due to " + fmt.Sprint(err))
	}

	// start threads
	(*ms.tm).StartThreads()
	(*ms.bc).StartThreads()

	err = api.InitMinerServer(config.Address, ms, ms.logger)
	if err != nil {
		panic("cannot init server twice!")
	}

	return ms
}
