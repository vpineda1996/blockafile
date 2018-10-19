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
	appendFee Balance
	createFee Balance
	opReward Balance
	noOpReward Balance
	accounts map[Account]Balance
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
	for k,v := range accUp {
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
	lg.Printf("Creating new blockchain state with %v as top", nd.Id)
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
		appendFee: Balance(appendFee),
		createFee: Balance(createFee),
		opReward: Balance(opReward),
		noOpReward: Balance(noOpReward),
		accounts: st,
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
			// TODO ksenia what should be first, award and then spend or vice-versa
			// award to miner
			award(res, Account(bae.Block.MinerId), opReward)

			// remove money for all involved accounts
			err := evaluateBalanceBlockOps(res, bae.Block.Records, appendFee, createFee)
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

// TODO EC1 delete: do something with block reward here
func evaluateBalanceBlockOps(accs map[Account]Balance, bcs []*crypto.BlockOp, appendFee Balance, createFee Balance) error {
	for _, tx := range bcs {
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
		default:
			return errors.New("Maria Magdalena (You're a victim of the fight You need love)")
		}
	}
	return nil
}

func spend(accs map[Account]Balance, act Account, fee Balance) error  {
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
	lg.Printf("Account %v got awarded %v, balance: %v", act, rw, accs[act])
}