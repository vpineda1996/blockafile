package state

import (
	"../../crypto"
	"../../shared/datastruct"
	"errors"
	"strconv"
)

type Account string
type Balance int

type AccountsState struct {
	appendFee  Balance
	createFee  Balance
	opReward   Balance
	noOpReward Balance
	accounts   map[Account]Balance
}

func (b AccountsState) GetAll() map[Account]Balance {
	return b.accounts
}

func (b AccountsState) GetAccountBalance(acc Account) Balance {
	if v, ok := b.accounts[acc]; ok {
		return v
	}
	return 0
}

func (b *AccountsState) update(accUp map[Account]Balance) {
	for k, v := range accUp {
		award(b.accounts, k, v)
	}
}

func NewAccountsState(
	appendFee int,
	createFee int,
	opReward int,
	noOpReward int,
	nd *datastruct.Node) (AccountsState, error) {
	if nd == nil {
		return AccountsState{
			accounts: make(map[Account]Balance),
		}, nil
	}
	lg.Printf("Creating new account state with %v reward and %v as top", opReward, nd.Id)
	nds := transverseChain(nd)
	st, err := generateState(
		Balance(appendFee),
		Balance(createFee),
		Balance(opReward),
		Balance(noOpReward),
		nds)
	if err != nil {
		return AccountsState{}, err
	}
	return AccountsState{
		appendFee:  Balance(appendFee),
		createFee:  Balance(createFee),
		opReward:   Balance(opReward),
		noOpReward: Balance(noOpReward),
		accounts:   st,
	}, nil
}

func generateState(
	appendFee Balance,
	createFee Balance,
	opReward Balance,
	noOpReward Balance,
	nodes []*datastruct.Node) (map[Account]Balance, error) {
	res := make(map[Account]Balance)

	// sanity checks
	if len(nodes) == 0 {
		return res, nil
	}
	switch nodes[0].Value.(type) {
	case crypto.BlockElement:
		if nodes[0].Value.(crypto.BlockElement).Block.Type != crypto.GenesisBlock {
			return nil, errors.New("genesis block should be the first block")
		}
	default:
		// if we reach this case then the tree is not built out of a blockchain, fail
		return nil, errors.New("cannot generate a state out of this blockchain")
	}

	// start iterating
	for idx, nd := range nodes {
		bae := nd.Value.(crypto.BlockElement)
		switch bae.Block.Type {
		case crypto.GenesisBlock:
			if idx != 0 {
				return nil, errors.New("genesis block should be the first block, not the " + strconv.Itoa(idx) + " block")
			}
			// do not award any currency to anybody
		case crypto.RegularBlock:
			// award to miner
			award(res, Account(bae.Block.MinerId), opReward)

			// remove money for all involved accounts
			err := evaluateBalanceBlockOps(res, bae.Block.Records, appendFee, createFee, nodes, idx)
			if err != nil {
				return nil, err
			}
		case crypto.NoOpBlock:
			// award to miner
			award(res, Account(bae.Block.MinerId), noOpReward)
		}
	}
	return res, nil
}

func evaluateBalanceBlockOps(accs map[Account]Balance, bcs []*crypto.BlockOp,
	appendFee Balance, createFee Balance, nds []*datastruct.Node, currBlockIdx int) error {
	for idx, tx := range bcs {
		switch tx.Type {
		case crypto.CreateFile:
			err := spend(accs, Account(tx.Creator), createFee)
			if err != nil {
				return err
			}
		case crypto.AppendFile:
			err := spend(accs, Account(tx.Creator), appendFee)
			if err != nil {
				return err
			}
		case crypto.DeleteFile:
			refund(accs, tx.Filename, appendFee, createFee, nds, currBlockIdx, idx)
		default:
			return errors.New("Maria Magdalena (You're a victim of the fight You need love)")
		}
	}
	return nil
}

// ASSUMPTION: ALL OF THE TRANSACTIONS IN THE BLOCK CHAIN ARE VALID FROM A FILESYSTEM PERSPECTIVE
func refund(accs map[Account]Balance, filename string, appendFee Balance, createFee Balance,
	nds []*datastruct.Node, currBlockIdx int, curTnxId int) {
	// start from most current until you see a delete
	// special handling for current block

	fnApplyTx := func(tx *crypto.BlockOp) bool {
		if tx.Filename == filename {
			if _, ok := accs[Account(tx.Creator)]; !ok {
				accs[Account(tx.Creator)] = 0
			}
			switch tx.Type {
			case crypto.CreateFile:
				lg.Printf("Refunding %v: %v ", tx.Creator, createFee)
				award(accs, Account(tx.Creator), createFee)
				return true
			case crypto.DeleteFile:
				return true
			case crypto.AppendFile:
				lg.Printf("Refunding %v: %v", tx.Creator, appendFee)
				award(accs, Account(tx.Creator), appendFee)
			}
		}
		return false
	}

	bae := nds[currBlockIdx].Value.(crypto.BlockElement).Block
	for j := curTnxId - 1; j >= 0; j-- {
		tx := bae.Records[j]
		if fnApplyTx(tx) {
			return
		}
	}

	// for the rest of the chain handle it as you should until you see a delete record
	for i := currBlockIdx - 1; i >= 0; i-- {
		bae := nds[i].Value.(crypto.BlockElement).Block
		if bae.Type != crypto.RegularBlock {
			continue
		}
		for j := len(bae.Records) - 1; j >= 0; j-- {
			tx := bae.Records[j]
			if fnApplyTx(tx) {
				return
			}
		}
	}
}

func spend(accs map[Account]Balance, act Account, fee Balance) error {
	lg.Printf("Account %v spent %v", act, fee)
	if v, ok := accs[act]; ok {
		if v >= fee {
			accs[act] -= fee
		} else {
			return errors.New("account " + string(act) + " has balance: " + strconv.Itoa(int(v)) +
				" but wanted to spend " + strconv.Itoa(int(fee)))
		}
		return nil
	} else {
		return errors.New("account " + string(act) + " wasn't found but wanted spend" + strconv.Itoa(int(fee)))
	}
}

func award(accs map[Account]Balance, act Account, rw Balance) {
	if _, ok := accs[act]; ok {
		accs[act] += rw
	} else {
		accs[act] = rw
	}
}
