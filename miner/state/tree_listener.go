package state

import (
	. "../../shared"
	"reflect"
)

type TreeListener interface {
	TreeEventHandler()
}

type AppendConfirmationListener struct {
	Creator string
	Filename string
	RecordNumber uint16
	Data [512]byte
	MinerState MinerState
	ConfirmsPerFileAppend int
	ConfirmsPerFileCreate int
	NotifyChannel chan int
}

func (acl AppendConfirmationListener) TreeEventHandler() {
	fs, err := acl.MinerState.GetFilesystemState(
		acl.ConfirmsPerFileCreate,
		acl.ConfirmsPerFileAppend)
	if err != nil {
		// todo ksenia what to do about this case?
		panic(err)
	}

	file, ok := fs.GetFile(Filename(acl.Filename))
	if !ok {
		return
	}
	if file.Creator != acl.Creator {
		return
	}
	if acl.RecordNumber >= file.NumberOfRecords {
		return
	}

	startIndex := acl.RecordNumber * 512
	if reflect.DeepEqual(acl.Data[:], file.Data[startIndex : startIndex + 512]) {
		acl.NotifyChannel <- 1
	}
	return
}

type CreateConfirmationListener struct {
	Creator string
	Filename string
	MinerState MinerState
	ConfirmsPerFileAppend int
	ConfirmsPerFileCreate int
	NotifyChannel chan int
}

func (ccl CreateConfirmationListener) TreeEventHandler() {
	fs, err := ccl.MinerState.GetFilesystemState(
		ccl.ConfirmsPerFileCreate,
		ccl.ConfirmsPerFileAppend)
	if err != nil {
		// todo ksenia what to do about this case?
		panic(err)
	}

	file, ok := fs.GetFile(Filename(ccl.Filename))
	if !ok {
		return
	}
	if file.Creator == ccl.Creator {
		ccl.NotifyChannel <- 1
	}
	return
}
