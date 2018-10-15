package api

import (
	"../../crypto"
	"../state"
	"errors"
)


type MinerServer struct {
	state *state.State
}

type EmptyArgs struct {}

type GetNodeArgs struct {
	id string
}

func (m *MinerServer) GetNode(args *GetNodeArgs, res *crypto.Block) error  {
	bk, ok := m.state.GetNode(args.id)
	if !ok {
		return errors.New("that node doesn't exist")
	}
	*res = *bk
	return nil
}

func (m *MinerServer) GetRoots(e *EmptyArgs, res *[]*crypto.Block) error  {
	bkArr := m.state.GetRoots()
	*res = bkArr
	return nil
}

func (m *MinerServer) GetOtherHosts(e *EmptyArgs, res *[]string) error  {
	return errors.New("not implemented")
}