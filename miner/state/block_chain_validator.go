package state

import (
	"../../crypto"
	. "../../shared"
	"errors"
	"fmt"
	"strconv"
)
import "../../shared/datastruct"


type BlockChainValidator struct {
	cnf                 Config
	mTree               *datastruct.MRootTree
	lastStateAccount    AccountsState
	lastFilesystemState FilesystemState

	generatingNodeId string
}

type FileAlreadyExistsValidationError struct {
	Filename string
}
func (e FileAlreadyExistsValidationError) Error() string {
	return "file " + e.Filename + " is duplicated, not a valid transaction"
}

type FileDoesNotExistValidationError struct {
	Filename string
}
func (e FileDoesNotExistValidationError) Error() string {
	return "file " + e.Filename + " doesn't exist but tried to append"
}

type AppendDuplicateValidationError struct {
	RecordNumber int
	FileName string
}
func (e AppendDuplicateValidationError) Error() string {
	return "append no " + strconv.Itoa(e.RecordNumber) +
		" to file " + e.FileName + " duplicated in chain, failing"
}

type MaxLengthReachedValidationError struct {
	FileName string
}
func (e MaxLengthReachedValidationError) Error() string {
	return fmt.Sprintf("file %s reached maximum length", e.FileName)
}

type NotEnoughMoneyValidationError struct {
	Account string
	ActualMoney int
	NeededMoney int
}
func (e NotEnoughMoneyValidationError) Error() string {
	return "balance for account " + e.Account + " is not enough, it has " + fmt.Sprintf("%v", e.ActualMoney) +
		" but it needs " + fmt.Sprintf("%v", e.NeededMoney)
}

// Given a block, it will return whether that block is valid or invalid
func (bcv *BlockChainValidator) Validate(b crypto.BlockElement) (*datastruct.Node, error) {
	// check if the current block is not present in the blockchain
	_, ok := bcv.mTree.Find(b.Id())
	if ok {
		return nil, errors.New("node is already on the blockchain")
	}

	valid := validateBlockHash(b, bcv.cnf.OpNumberOfZeros, bcv.cnf.NoOpNumberOfZeros)

	if !valid {
		return nil, errors.New("this is a corrupt node, failing")
	}

	if b.Block.Type == crypto.GenesisBlock {
		if len(bcv.mTree.GetRoots()) > 0 {
			return nil, errors.New("cannot add more than one genesis block")
		}
		return nil, nil
	}

	// get the prev block from the blockchain
	root, err := getParentNode(bcv.mTree, b.ParentId())
	if err != nil {
		return nil, err
	}

	// generate history if need be
	if bcv.generatingNodeId != root.Id {
		bcas, err := NewAccountsState(
			int(bcv.cnf.AppendFee),
			int(bcv.cnf.CreateFee),
			int(bcv.cnf.OpReward),
			int(bcv.cnf.NoOpReward),
			root)
		if err != nil {
			return nil, err
		}
		fss, err := NewFilesystemState(bcv.cnf.ConfirmsPerFileCreate, bcv.cnf.ConfirmsPerFileAppend, root)
		if err != nil {
			return nil, err
		}
		bcv.generatingNodeId = root.Id
		bcv.lastStateAccount = bcas
		bcv.lastFilesystemState = fss
	}

	fsUp, err := bcv.ValidateNewFSState(b)
	if err != nil {
		return nil, err
	}

	accUp, err := bcv.ValidateNewAccountState(b)
	if err != nil {
		return nil, err
	}

	bcv.generatingNodeId = b.Id()

	bcv.lastStateAccount.update(accUp)
	bcv.lastFilesystemState.update(fsUp)

	return root, nil
}


// TODO EC3 delete, do something here
func (bcv *BlockChainValidator) ValidateNewFSState(b crypto.BlockElement) (map[Filename]*FileInfo, error) {
	res := make(map[Filename]*FileInfo)
	bcs := b.Block.Records
	fs := bcv.lastFilesystemState.GetAll()
	for _, tx := range bcs {
		switch tx.Type {
		case crypto.CreateFile:
			if _, exists := fs[Filename(tx.Filename)]; exists {
				return nil, FileAlreadyExistsValidationError(tx.Filename)
			}

			if _, exists := res[Filename(tx.Filename)]; exists {
				return nil, FileAlreadyExistsValidationError(tx.Filename)
			}

			fi := FileInfo {
				Data:    make([]byte, 0, crypto.DataBlockSize),
				NumberOfRecords: 0,
				Creator: tx.Creator,
			}
			res[Filename(tx.Filename)] = &fi
		case crypto.AppendFile:
			if f, exists := fs[Filename(tx.Filename)]; exists {
				if f.NumberOfRecords >= MAX_RECORD_COUNT {
					return nil, MaxLengthReachedValidationError(tx.Filename)
				}

				newRecordNo := f.NumberOfRecords + 1
				if fi, inRes := res[Filename(tx.Filename)]; inRes {
					// ugly but we need it :(
					if tx.RecordNumber != fi.NumberOfRecords {
						return nil, AppendDuplicateValidationError{
							RecordNumber: int(tx.RecordNumber), FileName: tx.Filename}
					}
					newRecordNo = fi.NumberOfRecords + 1
				} else if tx.RecordNumber != f.NumberOfRecords {
					return nil, AppendDuplicateValidationError{
						RecordNumber: int(tx.RecordNumber), FileName: tx.Filename}
				}

				fi := FileInfo {
					Data:    make([]byte, 0, len(f.Data)),
					NumberOfRecords: newRecordNo,
					Creator: f.Creator,
				}
				res[Filename(tx.Filename)] = &fi
				copy(fi.Data, f.Data)
				lg.Printf("Adding record no %v to file %v", tx.RecordNumber, tx.Filename)
				fi.Data = append(fi.Data, FileData(tx.Data[:])...)
			} else {
				return nil, FileDoesNotExistValidationError(tx.Filename)
			}
		default:
			return nil, errors.New("invalid fs op")
		}
	}
	return res, nil
}

// TODO EC3 delete, do something here
func (bcv *BlockChainValidator) ValidateNewAccountState(b crypto.BlockElement) (map[Account]Balance, error) {
	res := make(map[Account]Balance)
	bcs := b.Block.Records
	accs := bcv.lastStateAccount

	// Award miner
	switch b.Block.Type {
	case crypto.NoOpBlock:
		award(res, Account(b.Block.MinerId), bcv.cnf.NoOpReward)
	case crypto.RegularBlock:
		award(res, Account(b.Block.MinerId), bcv.cnf.OpReward)
	default:
		return nil, errors.New("not a valid block type")
	}

	for _, tx := range bcs {
		act := Account(tx.Creator)
		var txFee Balance
		switch tx.Type {
		case crypto.CreateFile:
			txFee = bcv.cnf.CreateFee
		case crypto.AppendFile:
			txFee = bcv.cnf.AppendFee
		default:
			return nil, errors.New("not a valid file op")
		}

		// Verify miner has enough balance to perform transaction
		if _, ok := res[act]; !ok {
			res[act] = 0
		}
		if b := accs.GetAccountBalance(act) + res[act]; b < txFee {
			return nil, NotEnoughMoneyValidationError{Account: string(act), ActualMoney: int(b), NeededMoney: int(txFee)}
		}

		// Apply fee to the account
		res[act] -= txFee
	}
	return res, nil
}

func getParentNode(mTree *datastruct.MRootTree, id string) (*datastruct.Node, error)  {
	root, ok := mTree.Find(id)
	if !ok {
		return nil, errors.New("parent not in tree")
	}
	return root, nil
}

func validateBlockHash(b crypto.BlockElement, zerosOp int, zerosNoOp int) bool {
	return b.Block.Valid(zerosOp, zerosNoOp)
}

func NewBlockChainValidator(config Config, mTree *datastruct.MRootTree) *BlockChainValidator {
	return &BlockChainValidator{
		cnf:config,
		generatingNodeId: "",
		mTree: mTree,
	}
}

func transverseChain(root *datastruct.Node) []*datastruct.Node {
	res := make([]*datastruct.Node, root.Height + 1)
	// create list
	for nd, i := root, 0; nd != nil; nd, i = nd.Next(), i + 1 {
		res[i] = nd
	}
	// reverse
	for l, r := 0, len(res) - 1; l < r; l, r = l + 1, r - 1 {
		res[l], res[r] = res[r], res[l]
	}

	return res
}




