package api

import (
	"../../crypto"
	"errors"
	"github.com/DistributedClocks/GoVector/govec"
	"github.com/DistributedClocks/GoVector/govec/vrpc"
	"log"
	"net"
	"net/rpc"
)

type MinerServerListener interface {
	AddBlock(b *crypto.Block)
	AddJob(b crypto.BlockOp)
	AddHost(h string)
	GetBlock(id string) (*crypto.Block, bool)
	GetRoots() []*crypto.Block
}

type MinerServer struct {
	listener MinerServerListener
	logger   *govec.GoLog
}

type GetNodeArgs struct {
	Id string
	Host string
}

type GetNodeRes struct {
	Block crypto.Block
	Found bool
	Host string
}

// TODO SUPER IMPORTANT EVERY TIME THAT SOMEONE CALLS US ADD IT TO LIST OF CLIENTS!!!!!!

func (m *MinerServer) GetBlock(args *GetNodeArgs, res *GetNodeRes) error {
	m.listener.AddHost(args.Host)
	bk, ok := m.listener.GetBlock(args.Id)
	if ok {
		*res = GetNodeRes{
			Block: *bk,
			Found: ok,
		}
	} else {
		*res = GetNodeRes{
			Block: crypto.Block{},
			Found: ok,
		}
	}

	return nil
}

type EmptyArgs struct{
	Host string
}

func (m *MinerServer) GetRoots(e *EmptyArgs, res *[]*crypto.Block) error {
	m.listener.AddHost(e.Host)
	bkArr := m.listener.GetRoots()
	*res = bkArr
	return nil
}

func (m *MinerServer) GetOtherHosts(e *EmptyArgs, res *[]string) error {
	return errors.New("not implemented")
}

type ReceiveNodeArgs struct {
	Block crypto.Block
	Host string
}

func (m *MinerServer) ReceiveNode(args *ReceiveNodeArgs, res *bool) error {
	m.listener.AddHost(args.Host)
	*res = true
	m.listener.AddBlock(&args.Block)
	return nil
}

type ReceiveJobArgs struct {
	BlockOp crypto.BlockOp
	Host string
}

func (m *MinerServer) ReceiveJob(args *ReceiveJobArgs, res *bool) error {
	m.listener.AddHost(args.Host)
	*res = true
	m.listener.AddJob(args.BlockOp)
	return nil
}

func InitMinerServer(addr string, state MinerServerListener, logger *govec.GoLog) error {
	ms := &MinerServer{
		logger:   logger,
		listener: state,
	}
	server := rpc.NewServer()
	server.Register(ms)

	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("listen error:", e)
		return e
	}
	lg.Printf("Started listening on port: %v", addr)
	go vrpc.ServeRPCConn(server, l, logger, govec.GetDefaultLogOptions())
	return nil
}
