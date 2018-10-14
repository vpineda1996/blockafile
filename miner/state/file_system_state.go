package state

import (
	"../../crypto"
	"../../shared/datastruct"
	"errors"
	"strconv"
)

type Filename string
type FileData []byte
type FileInfo struct {
	Creator string
	NumberOfRecords uint32
	Data    FileData
}

type FilesystemState struct {
	fs map[Filename]*FileInfo
}

func (b FilesystemState) GetAll() map[Filename]*FileInfo {
	return b.fs
}

func (b FilesystemState) GetFile(acc Filename) (*FileInfo, bool) {
	v, ok := b.fs[acc]
	return v, ok
}

func NewFilesystemState(nd *datastruct.Node) (FilesystemState, error) {
	if nd == nil {
		return FilesystemState{
			fs: make(map[Filename]*FileInfo),
		}, nil
	}
	lg.Printf("Creating new fs state with %v as top", nd.NodeId)
	nds := transverseChain(nd)
	fs, err := generateFilesystem(nds)
	return FilesystemState{
		fs:fs,
	}, err
}

func generateFilesystem(nodes []*datastruct.Node) (map[Filename]*FileInfo, error) {
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
			err := evaluateFSBlockOps(res, bae.Block.Records)
			if err != nil {
				return nil, err
			}
		case crypto.NoOpBlock:
			// do nothing here
		}
	}
	return res, nil
}
// TODO EC1 delete add the case over here to add the record
func evaluateFSBlockOps(fs map[Filename]*FileInfo, bcs []*crypto.BlockOp ) error {
	for _, tx := range bcs {
		switch tx.Type {
		case crypto.CreateFile:
			if _, exists := fs[Filename(tx.Filename)]; exists {
				return errors.New("file " + tx.Filename + " is duplicated, not a valid transaction")
			}
			fi := FileInfo {
				Data:    make([]byte, 0, crypto.DataBlockSize),
				NumberOfRecords: 0,
				Creator: tx.Creator,
			}
			fs[Filename(tx.Filename)] = &fi
		case crypto.AppendFile:
			if f, exists := fs[Filename(tx.Filename)]; exists {
				if tx.RecordNumber != f.NumberOfRecords {
					return errors.New("append no " + strconv.Itoa(int(tx.RecordNumber)) +
						" to file " + tx.Filename + " duplicated in chain, failing")
				}
				f.NumberOfRecords += 1
				f.Data = append(f.Data, FileData(tx.Data[:])...)
				return nil
			}
			return errors.New("file " + tx.Filename + " doesn't exist but tried to append")
		default:
			return errors.New("vous les hommes êtes tous les mêmes, Macho mais cheap, Bande de mauviettes infidèles")
		}
	}
	return nil
}