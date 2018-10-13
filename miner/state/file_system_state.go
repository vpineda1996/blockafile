package state

import (
	"../../shared/datastruct"
)

type Filename string
type Filedata []byte

type FilesystemState struct {
	accounts map[Filename]Filedata
}

func NewFilesystemState(nd *datastruct.Node) FilesystemState {
	return FilesystemState{}
}

// TODO Given a node (ie the top of the largest chain), it will generate a state
// TODO of the filesystem up to that node