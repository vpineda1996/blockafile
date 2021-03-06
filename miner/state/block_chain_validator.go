package state

import (
	"../../crypto"
	. "../../shared"
	"errors"
	"fmt"
	"strconv"
	"sync"
)
import "../../shared/datastruct"

type BlockChainValidator struct {
	cnf                 Config
	mTree               *datastruct.MRootTree
	lastStateAccount    AccountsState
	lastFilesystemState FilesystemState
	mtx                 *sync.Mutex

	generatingNodeId string
}

type BlockChainValidatorError interface{
	GetErrorCode() FailureType
	Error() string
}

type FileAlreadyExistsValidationError struct {
	Filename string
}

func (e FileAlreadyExistsValidationError) GetErrorCode() FailureType {
	return FILE_EXISTS
}

func (e FileAlreadyExistsValidationError) Error() string {
	return "file " + e.Filename + " is duplicated, not a valid transaction"
}

type FileDoesNotExistValidationError struct {
	Filename string
}

func (e FileDoesNotExistValidationError) GetErrorCode() FailureType {
	return FILE_DOES_NOT_EXIST
}

func (e FileDoesNotExistValidationError) Error() string {
	return "file " + e.Filename + " doesn't exist but tried to append"
}

type AppendDuplicateValidationError struct {
	RecordNumber int
	FileName string
}

func (e AppendDuplicateValidationError) GetErrorCode() FailureType {
	return APPEND_DUPLICATE
}

func (e AppendDuplicateValidationError) Error() string {
	return "append no " + strconv.Itoa(e.RecordNumber) +
		" to file " + e.FileName + " duplicated in chain, failing"
}

type MaxLengthReachedValidationError struct {
	FileName string
}

func (e MaxLengthReachedValidationError) GetErrorCode() FailureType {
	return MAX_LEN_REACHED
}

func (e MaxLengthReachedValidationError) Error() string {
	return fmt.Sprintf("file %s reached maximum length", e.FileName)
}

type BadFileNameValidationError struct {
	FileName string
}

func (e BadFileNameValidationError) GetErrorCode() FailureType {
	return BAD_FILENAME
}

func (e BadFileNameValidationError) Error() string {
	return fmt.Sprintf("file %s has filename that is too long", e.FileName)
}

type NotEnoughMoneyValidationError struct {
	Account string
	ActualMoney int
	NeededMoney int
}

func (e NotEnoughMoneyValidationError) GetErrorCode() FailureType {
	return NOT_ENOUGH_MONEY
}

func (e NotEnoughMoneyValidationError) Error() string {
	return "balance for account " + e.Account + " is not enough, it has " + fmt.Sprintf("%v", e.ActualMoney) +
		" but it needs " + fmt.Sprintf("%v", e.NeededMoney)
}

type UnspecifiedValidationError string

func (e UnspecifiedValidationError) GetErrorCode() FailureType {
	return NO_ERROR
}

func (e UnspecifiedValidationError) Error() string {
	return string(e)
}

type CompositeError struct {
	Prev BlockChainValidatorError
	Current BlockChainValidatorError
}

func (e CompositeError) Error() string {
	return fmt.Sprintln(e.Prev) + fmt.Sprintln(e.Current)
}

func (e CompositeError) GetErrorCode() FailureType {
	if e.Current != nil {
		return e.Current.GetErrorCode()
	}
	if e.Prev != nil {
		return e.Prev.GetErrorCode()
	}
	return NO_ERROR
}

// Given a block, it will return whether that block is valid or invalid
func (bcv *BlockChainValidator) Validate(b crypto.BlockElement) (*datastruct.Node, error) {
	// check if the current block is not present in the blockchain
	_, ok := bcv.mTree.Find(b.Id())
	if ok {
		return nil, errors.New("node is already on the blockchain")
	}

	if b.Block.Type == crypto.GenesisBlock {
		if len(bcv.mTree.GetRoots()) > 0 {
			return nil, errors.New("cannot add more than one genesis block")
		}
		return nil, nil
	}

	valid := validateBlockHash(b, bcv.cnf.OpNumberOfZeros, bcv.cnf.NoOpNumberOfZeros)

	if !valid {
		return nil, errors.New("this is a corrupt node, failing")
	}

	// get the prev block from the blockchain
	root, err := getParentNode(bcv.mTree, b.ParentId())
	if err != nil {
		return nil, err
	}

	bcv.mtx.Lock()
	defer bcv.mtx.Unlock()
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

	fsUp, deletedFiles, err := bcv.validateNewFSState(b)
	if err != nil {
		return nil, err
	}

	accUp, err := bcv.validateNewAccountState(b, root.Id)
	if err != nil {
		return nil, err
	}

	bcv.generatingNodeId = b.Id()

	bcv.lastStateAccount.update(accUp)
	bcv.lastFilesystemState.update(fsUp, deletedFiles)

	return root, nil
}


func (bcv *BlockChainValidator) ValidateJobSet(
	ops []*crypto.BlockOp,
	rootNode *datastruct.Node) (newOps []*crypto.BlockOp, accountsError error, filesError error) {
	if len(ops) == 0 {
		return ops, nil, nil
	}

	bcv.mtx.Lock()
	defer bcv.mtx.Unlock()

	if bcv.generatingNodeId != rootNode.Id {
		bcas, err := NewAccountsState(
			int(bcv.cnf.AppendFee),
			int(bcv.cnf.CreateFee),
			int(bcv.cnf.OpReward),
			int(bcv.cnf.NoOpReward),
			rootNode)
		if err != nil {
			return []*crypto.BlockOp{}, err, nil
		}
		fss, err := NewFilesystemState(bcv.cnf.ConfirmsPerFileCreate, bcv.cnf.ConfirmsPerFileAppend, rootNode)
		if err != nil {
			return []*crypto.BlockOp{}, nil, err
		}
		bcv.generatingNodeId = rootNode.Id
		bcv.lastStateAccount = bcas
		bcv.lastFilesystemState = fss
	}

	newOps, original := ops, -1
	for original != len(newOps) {
		original = len(newOps)
		nFile := make(map[Filename]*FileInfo)
		var err error
		newOps, _, err = bcv.validateNewFSBlockOps(newOps, nFile)
		if err != nil {
			filesError = err
			lg.Printf("Rejected some ops, the following is a sample error: %v\n", err)
		}

		nAcc := make(map[Account]Balance)
		newOps, err = bcv.validateNewAccountBlockOps(newOps, bcv.mTree.GetLongestChain().Id, nAcc)
		if err != nil {
			accountsError = err
			lg.Printf("Rejected some ops, the following is a sample error: %v\n", err)
		}
	}
	return newOps, accountsError, filesError
}

func (bcv *BlockChainValidator) validateNewFSState(b crypto.BlockElement) (map[Filename]*FileInfo, map[string]bool, error) {
	res := make(map[Filename]*FileInfo)
	bcs := b.Block.Records
	_, deletedFiles, err := bcv.validateNewFSBlockOps(bcs, res)
	return res, deletedFiles, err
}

func (bcv *BlockChainValidator) validateNewFSBlockOps(bcs []*crypto.BlockOp,
		res map[Filename]*FileInfo) ([]*crypto.BlockOp, map[string]bool, error) {
	validOps := make([]*crypto.BlockOp, 0, len(bcs))
	deletedFiles := make(map[string]bool)
	var err BlockChainValidatorError = nil
	fs := bcv.lastFilesystemState.GetAll()
	for _, tx := range bcs {
		switch tx.Type {
		case crypto.CreateFile:
			if len(tx.Filename) > MAX_FILENAME_LENGTH {
				err = CompositeError{err, BadFileNameValidationError{tx.Filename}}
				continue
			}

			if _, exists := fs[Filename(tx.Filename)]; exists {
				if _, deleted := deletedFiles[tx.Filename]; !deleted {
					err = CompositeError{
						err,
						FileAlreadyExistsValidationError{tx.Filename}}
					continue
				}
			}

			if _, exists := res[Filename(tx.Filename)]; exists {
				if _, deleted := deletedFiles[tx.Filename]; !deleted {
					err = CompositeError {
						err,
						FileAlreadyExistsValidationError{tx.Filename}}
					continue
				}
			}
			lg.Printf("validator: creating file %v", tx.Filename)
			fi := FileInfo{
				Data:            make([]byte, 0, crypto.DataBlockSize),
				NumberOfRecords: 0,
				Creator:         tx.Creator,
			}
			res[Filename(tx.Filename)] = &fi
			validOps = append(validOps, tx)
			if _, deleted := deletedFiles[tx.Filename]; deleted {
				delete(deletedFiles, tx.Filename)
			}
		case crypto.AppendFile:
			// check if the file is deleted, if it is make this tnx invalid
			if _, deleted := deletedFiles[tx.Filename]; deleted {
				err = CompositeError {
					err,
					FileDoesNotExistValidationError{tx.Filename}}
				continue
			}

			// otherwise, proceed with append
			if f, exists := fs[Filename(tx.Filename)]; exists {
				if f.NumberOfRecords >= MAX_RECORD_COUNT {
					err = CompositeError {
						err,
						MaxLengthReachedValidationError{tx.Filename}}
					continue
				}

				newRecordNo := f.NumberOfRecords + 1
				if fi, inRes := res[Filename(tx.Filename)]; inRes {
					// ugly but we need it :(
					if tx.RecordNumber != fi.NumberOfRecords {
						err = CompositeError {
							err,
							AppendDuplicateValidationError{
								int(tx.RecordNumber),
								tx.Filename}}
						continue
					}
					newRecordNo = fi.NumberOfRecords + 1
				} else if tx.RecordNumber != f.NumberOfRecords {
					err = CompositeError {
						err,
						AppendDuplicateValidationError{
							int(tx.RecordNumber),
							tx.Filename}}
					continue
				}

				fi := FileInfo{
					Data:            make([]byte, 0, len(f.Data)),
					NumberOfRecords: newRecordNo,
					Creator:         f.Creator,
				}
				res[Filename(tx.Filename)] = &fi
				copy(fi.Data, f.Data)
				lg.Printf("Adding record no %v to file %v", tx.RecordNumber, tx.Filename)
				fi.Data = append(fi.Data, FileData(tx.Data[:])...)
				validOps = append(validOps, tx)
			} else if donkey, inRes := res[Filename(tx.Filename)]; inRes {
				if tx.RecordNumber != donkey.NumberOfRecords {
					err = CompositeError {
						err,
						AppendDuplicateValidationError{
							int(tx.RecordNumber),
							tx.Filename}}
					continue
				}
				monkey := FileInfo{
					Data:            make([]byte, 0, len(donkey.Data)),
					NumberOfRecords: donkey.NumberOfRecords + 1,
					Creator:         donkey.Creator,
				}
				res[Filename(tx.Filename)] = &monkey
				copy(monkey.Data, donkey.Data)
				lg.Printf("Adding record no %v to file %v", tx.RecordNumber, tx.Filename)
				monkey.Data = append(monkey.Data, FileData(tx.Data[:])...)
				validOps = append(validOps, tx)
			} else {
				err = CompositeError {
					err,
					FileDoesNotExistValidationError{tx.Filename}}
				continue
			}
		case crypto.DeleteFile:
			// super easy, when we delete a file most of the hard work will be done by create, update and append
			if _, deleted := deletedFiles[tx.Filename]; deleted {
				err = CompositeError {
					err,
					FileDoesNotExistValidationError{tx.Filename}}
				continue
			}
			if _, inRes := res[Filename(tx.Filename)]; !inRes {
				if _, exists := fs[Filename(tx.Filename)]; !exists {
					// todo add error types for this once there is client support for this
					err = CompositeError {
						err,
						FileDoesNotExistValidationError{tx.Filename}}
					continue
				}
			}
			lg.Printf("validator: removing file %v", tx.Filename)
			validOps = append(validOps, tx)
			deletedFiles[tx.Filename] = true
		default:

			err = CompositeError {
				err,
				UnspecifiedValidationError("invalid fs op")}
			continue
		}
	}
	return validOps, deletedFiles, err
}

func (bcv *BlockChainValidator) validateNewAccountState(b crypto.BlockElement, parentBlock string) (map[Account]Balance, error) {
	res := make(map[Account]Balance)
	bcs := b.Block.Records

	// Award miner
	switch b.Block.Type {
	case crypto.NoOpBlock:
		award(res, Account(b.Block.MinerId), bcv.cnf.NoOpReward)
	case crypto.RegularBlock:
		award(res, Account(b.Block.MinerId), bcv.cnf.OpReward)
	default:
		return nil, errors.New("not a valid block type")
	}

	_, err := bcv.validateNewAccountBlockOps(bcs, parentBlock, res)
	return res, err
}

func (bcv *BlockChainValidator) validateNewAccountBlockOps(bcs []*crypto.BlockOp, parentBlock string, res map[Account]Balance) ([]*crypto.BlockOp, error) {
	accs := bcv.lastStateAccount
	validOps := make([]*crypto.BlockOp, 0, len(bcs))
	var err BlockChainValidatorError = nil
	for idx, tx := range bcs {
		act := Account(tx.Creator)
		var txFee Balance
		switch tx.Type {
		case crypto.CreateFile:
			txFee = bcv.cnf.CreateFee
		case crypto.AppendFile:
			txFee = bcv.cnf.AppendFee
		case crypto.DeleteFile:
			// stupidly expensive way of doing this, better options?
			parent, ok := bcv.mTree.Find(parentBlock)
			if !ok {
				// todo add error types for this once there is client support for this
				err = CompositeError{
					err,
					UnspecifiedValidationError("coudn't find parent block to calculate refund")}
				continue
			}

			// create node chain
			nds := transverseChain(parent)

			// fake block to make things work
			fakeBlock := &crypto.Block{
				Type: crypto.RegularBlock,
				Records: bcs,
			}
			fakeNode := datastruct.Node{
				Value: crypto.BlockElement{
					Block: fakeBlock,
				},
			}
			nds = append(nds, &fakeNode)
			refund(res, tx.Filename, bcv.cnf.AppendFee, bcv.cnf.CreateFee, nds, len(nds) - 1, idx)
			validOps = append(validOps, tx)
			continue
		default:
			return []*crypto.BlockOp{}, errors.New("not a valid file op")
		}

		// Verify miner has enough balance to perform transaction
		if _, ok := res[act]; !ok {
			res[act] = 0
		}
		if b := accs.GetAccountBalance(act) + res[act]; b < txFee {
			err = CompositeError{
				err,
				NotEnoughMoneyValidationError{string(act), int(b), int(txFee)}}
			continue
		} else {
			// Apply fee to the account
			res[act] -= txFee
			validOps = append(validOps, tx)
		}
	}
	return validOps, err
}

func getParentNode(mTree *datastruct.MRootTree, id string) (*datastruct.Node, error) {
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
		cnf:              config,
		generatingNodeId: "",
		mTree:            mTree,
		mtx:              new(sync.Mutex),
	}
}

func transverseChain(root *datastruct.Node) []*datastruct.Node {
	res := make([]*datastruct.Node, root.Height+1)
	// create list
	for nd, i := root, 0; nd != nil; nd, i = nd.Next(), i+1 {
		res[i] = nd
	}
	// reverse
	for l, r := 0, len(res)-1; l < r; l, r = l+1, r-1 {
		res[l], res[r] = res[r], res[l]
	}

	return res
}
