package state

import (
	"../../shared/tree"
	"net/rpc"
	"sync"
)

type TreeManager struct {
	clients []*rpc.Client
	mTree *tree.MRootTree
	mtx *sync.Mutex
}

func (t *TreeManager) AddClient(c *rpc.Client) {
	t.clients = append(t.clients, c)
}

// TODO vpineda create 3 go routines

func New() *TreeManager {
	return &TreeManager{
		mtx: new(sync.Mutex),
		clients: make([]*rpc.Client, 0, 1),
		mTree: tree.NewMRootTree(),
	}
}