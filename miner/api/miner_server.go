package api

import (
	"../../crypto"
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
)


type MinerServerListener interface {
	AddBlock(b *crypto.Block)
	AddJob(b *crypto.BlockOp)
	GetBlock(id string) (*crypto.Block, bool)
	GetRoots() []*crypto.Block
}

type MinerServer struct {
	listener MinerServerListener
}

type GetNodeArgs struct {
	Id string
}

type GetNodeRes struct {
	Block crypto.Block
	Found bool
}

func (m *MinerServer) GetBlock(args *GetNodeArgs, res *GetNodeRes) error  {
	bk, ok := m.listener.GetBlock(args.Id)
	*res = GetNodeRes{
		Block: *bk,
		Found: ok,
	}
	return nil
}

type EmptyArgs struct {}

func (m *MinerServer) GetRoots(e *EmptyArgs, res *[]*crypto.Block) error  {
	bkArr := m.listener.GetRoots()
	*res = bkArr
	return nil
}

func (m *MinerServer) GetOtherHosts(e *EmptyArgs, res *[]string) error  {
	return errors.New("not implemented")
}


type ReceiveNodeArgs struct {
	Block crypto.Block
}

func (m *MinerServer) ReceiveNode(args *ReceiveNodeArgs, res *bool) error {
	*res = true
	m.listener.AddBlock(&args.Block)
	return nil
}

type ReceiveJobArgs struct {
	BlockOp crypto.BlockOp
}

func (m *MinerServer) ReceiveJob(args *ReceiveJobArgs, res *bool) error {
	*res = true
	m.listener.AddJob(&args.BlockOp)
	return nil
}

func InitMinerServer(addr string, state MinerServerListener) error {
	ms := new(MinerServer)
	ms.listener = state
	rpc.Register(ms)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("listen error:", e)
		return e
	}
	go http.Serve(l, nil)
	return nil
}