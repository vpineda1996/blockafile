package state

import (
	"../../crypto"
	"../../shared"
	"../api"
	. "../block_calculators"
	"container/list"
	"crypto/md5"
	"fmt"
	"github.com/DistributedClocks/GoVector/govec"
	"log"
	"os"
	"sync"
	"time"
)

type MinerState struct {
	// dont ask of the double ptr, its cancer but there is no other way
	logger    *govec.GoLog
	tm        **TreeManager
	clients   *map[string]*api.MinerClient
	clientsMux *sync.Mutex
	bc        **BlockCalculator
	minerId   string
	outgoingIP string
	incomingAddr string
	listeners *list.List
	listenersMux *sync.Mutex
	singleMinerDisconnected bool
}

type Config struct {
	AppendFee             Balance // Note that this is not user-configured. Always exactly 1 coin.
	CreateFee             Balance
	OpReward              Balance
	NoOpReward            Balance
	OpNumberOfZeros       int
	NoOpNumberOfZeros	  int
	OutgoingMinersIP  	  string
	IncomingMinersAddr	  string
	ConfirmsPerFileCreate int
	ConfirmsPerFileAppend int
	OpPerBlock            int
	MinerId               string
	GenesisBlockHash      [md5.Size]byte
	GenOpBlockTimeout     uint8
	SingleMinerDisconnected bool // true if we consider a single miner to be 'disconnected' from the network
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
	cpyClients := make(map[string]*api.MinerClient)

	s.clientsMux.Lock()
	for k, v := range *s.clients {
		cpyClients[k] = v
	}
	s.clientsMux.Unlock()

	for _, c := range cpyClients {
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
	cpyClients := make(map[string]*api.MinerClient)

	s.clientsMux.Lock()
	for k, v := range *s.clients {
		cpyClients[k] = v
	}
	s.clientsMux.Unlock()

	for k, c := range cpyClients {
		arr, err := c.GetRoots()
		if err != nil {
			lg.Printf("error in connection node %v\n", err)
			s.clientsMux.Lock()
			delete(*s.clients, k)
			s.clientsMux.Unlock()
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
	//s.logger.LogLocalEvent(fmt.Sprintf(" Block %v added to tree", b.Id()), INFO)
	(*s.bc).RemoveJobsFromBlock(b)
}

func (s MinerState) OnNewBlockInLongestChain(b *crypto.Block) {
	s.listenersMux.Lock()
	defer s.listenersMux.Unlock()
	for e := s.listeners.Front(); e != nil; e = e.Next() {
		go func(node *list.Element) {
			if node != nil && node.Value != nil {
				if succeed := node.Value.(TreeListener).TreeEventHandler(); succeed {
					s.listenersMux.Lock()
					s.listeners.Remove(node)
					s.listenersMux.Unlock()
				} else if expired := node.Value.(TreeListener).IsExpired(); expired {
					s.listenersMux.Lock()
					s.listeners.Remove(node)
					s.listenersMux.Unlock()
				}
			}
		}(e)
	}
	//s.logger.LogLocalEvent(fmt.Sprintf(" New head on longest chain: %v", b.Id()), INFO)
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
		lg.Printf("WARN: Recieved block %v but rejected", b.Id())
		//s.logger.LogLocalEvent(fmt.Sprintf(" Recieved block %v but I have it", b.Id()), WARN)
	}
}

func (s MinerState) broadcastBlock(b *crypto.Block) {
	go func() {
		cpyClients := make(map[string]*api.MinerClient)
		s.clientsMux.Lock()
		for k, v := range *s.clients {
			cpyClients[k] = v
		}
		s.clientsMux.Unlock()

		for k, c := range cpyClients {
			lg.Printf("Sending block to: %v", k)
			c.SendBlock(b)
		}
	}()
}

func (s MinerState) AddJob(b crypto.BlockOp) {
	if (*s.bc).JobExists(&b) < 0 {
		lg.Printf("Added new job: %v", b.Filename)
		//s.logger.LogLocalEvent(fmt.Sprintf(" Enqueuing job for file %v and record %v for miner to work on", b.Filename, b.RecordNumber), INFO)
		(*s.bc).AddJob(&b)
		s.broadcastJob(&b)
	} else {
		lg.Printf("WARN: Recieved job for file %v but rejected", b.Filename)
		//s.logger.LogLocalEvent(fmt.Sprintf(" Recieved job for file %v but I have it", b.Filename), WARN)
	}

}

func (s MinerState) broadcastJob(b *crypto.BlockOp) {
	go func() {

		cpyClients := make(map[string]*api.MinerClient)
		s.clientsMux.Lock()
		for k, v := range *s.clients {
			cpyClients[k] = v
		}
		s.clientsMux.Unlock()

		for k, c := range cpyClients {
			lg.Printf("Sending job to: %v", k)
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

func (s MinerState) ValidateJobSet(bOps []*crypto.BlockOp) ([]*crypto.BlockOp, error, error) {
	return (*s.tm).ValidateJobSet(bOps)
}

func (s MinerState) InLongestChain(id string) int {
	return (*s.tm).InLongestChain(id)
}

func (s MinerState) SleepMiner() {
	(*s.bc).ShutdownThreads()
}

func (s MinerState) ActivateMiner(){
	(*s.bc).StartThreads()
}

func (s MinerState) AddHost(h string) {
	s.clientsMux.Lock()
	if _, ok := (*s.clients)[h]; !ok {
		conn, err := api.NewMinerClient(h, s.incomingAddr, s.outgoingIP, s.logger)
		if err == nil {
			(*s.clients)[h] = &conn
		} else {
			lg.Printf("Couldn't connect to %v due to %v", h, err)
		}
	}
	s.clientsMux.Unlock()
}

func (s MinerState) AddTreeListener(listener TreeListener) {
	s.listenersMux.Lock()
	s.listeners.PushBack(listener)
	s.listenersMux.Unlock()
}

func (s MinerState) IsDisconnected() bool {
	return s.singleMinerDisconnected && len(*s.clients) == 0
}

func NewMinerState(config Config, connectedMiningNodes []string) MinerState {
	logger := govec.InitGoVector(config.MinerId, shared.LOGFILE + "_" + config.MinerId, shared.GoVecOpts)
	cls := make(map[string]*api.MinerClient, len(connectedMiningNodes))
	for _, c := range connectedMiningNodes {
		conn, err := api.NewMinerClient(c, config.IncomingMinersAddr, config.OutgoingMinersIP, logger)
		if err == nil {
			cls[c] = &conn
		} else {
			lg.Printf("Couldn't connect to %v due to %v", c, err)
		}
	}
	var treePtr *TreeManager
	var blockCalcPtr *BlockCalculator
	ms := MinerState{
		clients:   &cls,
		clientsMux: new(sync.Mutex),
		minerId:   config.MinerId,
		tm:        &treePtr,
		bc:        &blockCalcPtr,
		logger:    logger,
		outgoingIP: config.OutgoingMinersIP,
		incomingAddr: config.IncomingMinersAddr,
		listeners: list.New(),
		listenersMux: new(sync.Mutex),
		singleMinerDisconnected: config.SingleMinerDisconnected,
	}
	treePtr = NewTreeManager(config, ms, ms)

	calcThresh := config.ConfirmsPerFileCreate
	if config.ConfirmsPerFileAppend > calcThresh {
		calcThresh = config.ConfirmsPerFileAppend
	}

	blockCalcPtr = NewBlockCalculator(ms,
		config.OpNumberOfZeros,
		config.NoOpNumberOfZeros,
		config.OpPerBlock,
		time.Duration(config.GenOpBlockTimeout),
		calcThresh)

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

	err = api.InitMinerServer(config.IncomingMinersAddr, ms, ms.logger)
	if err != nil {
		panic("cannot init server twice!")
	}

	return ms
}
