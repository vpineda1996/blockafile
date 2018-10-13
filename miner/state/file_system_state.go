package state

import (
	"../../shared/datastruct"
)


type FilesystemState struct {
	accounts map[string]int
}

func NewFilesystemState(nd *datastruct.Node) FilesystemState {
	return FilesystemState{}
}

// TODO Given a node (ie the top of the largest chain), it will generate a state
// TODO of the filesystem up to that node