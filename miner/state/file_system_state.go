package state

import (
	"../../crypto"
	. "../../shared"
	"../../shared/datastruct"
	"errors"
	"strconv"
)

type FilesystemState struct {
	fs map[Filename]*FileInfo
}

func (b FilesystemState) GetAll() map[Filename]*FileInfo {
	return b.fs
}

func (b *FilesystemState) update(newData map[Filename]*FileInfo, deletedFiles map[string]bool) {
	for k, v := range newData {
		b.fs[k] = v
	}
	for k := range deletedFiles {
		delete(b.fs, Filename(k))
	}
}

func (b FilesystemState) GetFile(acc Filename) (*FileInfo, bool) {
	v, ok := b.fs[acc]
	return v, ok
}

func NewFilesystemState(
	confirmsPerFileCreate int,
	confirmsPerFileAppend int,
	nd *datastruct.Node) (FilesystemState, error) {
	if nd == nil {
		return FilesystemState{
			fs: make(map[Filename]*FileInfo),
		}, nil
	}
	lg.Printf("Creating new fs state with %v as top", nd.Id)
	nds := transverseChain(nd)
	fs, err := generateFilesystem(nds, confirmsPerFileCreate, confirmsPerFileAppend)

	return FilesystemState{
		fs: fs,
	}, err
}

func generateFilesystem(
	nodes []*datastruct.Node,
	confirmsPerFileCreate int,
	confirmsPerFileAppend int) (map[Filename]*FileInfo, error) {
	res := make(map[Filename]*FileInfo)

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
			createOpsConfirmed := false
			appendOpsConfirmed := false
			numNodesInFrontOfMe := len(nodes) - idx - 1
			if numNodesInFrontOfMe >= confirmsPerFileCreate {
				createOpsConfirmed = true
			}
			if numNodesInFrontOfMe >= confirmsPerFileAppend {
				appendOpsConfirmed = true
			}
			err := evaluateFSBlockOps(res, bae.Block.Records, createOpsConfirmed, appendOpsConfirmed)
			if err != nil {
				return nil, err
			}
		case crypto.NoOpBlock:
			// do nothing here
		}
	}
	return res, nil
}

func evaluateFSBlockOps(
	fs map[Filename]*FileInfo,
	bcs []*crypto.BlockOp,
	createOpsConfirmed bool,
	appendOpsConfirmed bool) error {
	for _, tx := range bcs {
		switch tx.Type {
		case crypto.CreateFile:
			if createOpsConfirmed {
				if len(tx.Filename) > MaxFileName {
					return errors.New("filename is to big for the given file")
				}

				if _, exists := fs[Filename(tx.Filename)]; exists {
					return errors.New("file " + tx.Filename + " is duplicated, not a valid transaction")
				}
				lg.Printf("Creating file %v", tx.Filename)
				fi := FileInfo{
					Data:            make([]byte, 0, crypto.DataBlockSize),
					NumberOfRecords: 0,
					Creator:         tx.Creator,
				}
				fs[Filename(tx.Filename)] = &fi
			}
		case crypto.AppendFile:
			if appendOpsConfirmed {
				lg.Printf("Appending to file %v record no %v", tx.Filename, tx.RecordNumber)
				if f, exists := fs[Filename(tx.Filename)]; exists {
					if tx.RecordNumber != f.NumberOfRecords {
						return errors.New("append no " + strconv.Itoa(int(tx.RecordNumber)) +
							" to file " + tx.Filename + " duplicated in chain, failing")
					}
					f.NumberOfRecords += 1
					f.Data = append(f.Data, FileData(tx.Data[:])...)
				} else {
					return errors.New("file " + tx.Filename + " doesn't exist but tried to append")
				}
			}
		case crypto.DeleteFile:
			lg.Printf("in delete")
			if createOpsConfirmed {
				if _, exists := fs[Filename(tx.Filename)]; !exists {
					return errors.New("file " + tx.Filename + " doesn't exist and cannot delete")
				}
				lg.Printf("Deleting file %v", tx.Filename)
				delete(fs, Filename(tx.Filename))
			}
		default:
			return errors.New("vous les hommes êtes tous les mêmes, Macho mais cheap, Bande de mauviettes infidèles")
		}
	}
	return nil
}
