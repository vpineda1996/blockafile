package state

import (
	. "../../shared"
	"bytes"
	"time"
)

type TreeListener interface {
	TreeEventHandler() bool
	IsExpired() bool
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
	ExpirationTime time.Time
}

func (acl AppendConfirmationListener) TreeEventHandler() bool {
	fs, err := acl.MinerState.GetFilesystemState(
		acl.ConfirmsPerFileCreate,
		acl.ConfirmsPerFileAppend)
	if err != nil {
		lg.Println("AppendConfirmationListener, ", err)
		return false
	}

	file, ok := fs.GetFile(Filename(acl.Filename))
	if !ok {
		return false
	}
	if file.Creator != acl.Creator {
		return false
	}
	if acl.RecordNumber >= file.NumberOfRecords {
		return false
	}

	startIndex := uint32(acl.RecordNumber) * 512
	if bytes.Equal(acl.Data[:], file.Data[startIndex : startIndex + 512]) {
		acl.NotifyChannel <- 1
		return true
	}
	return false
}

func (acl AppendConfirmationListener) IsExpired() bool {
	return isPastTime(acl.ExpirationTime)
}

type CreateConfirmationListener struct {
	Creator string
	Filename string
	MinerState MinerState
	ConfirmsPerFileAppend int
	ConfirmsPerFileCreate int
	NotifyChannel chan int
	ExpirationTime time.Time
}

func (ccl CreateConfirmationListener) TreeEventHandler() bool {
	fs, err := ccl.MinerState.GetFilesystemState(
		ccl.ConfirmsPerFileCreate,
		ccl.ConfirmsPerFileAppend)
	if err != nil {
		lg.Println("CreateConfirmationListener, ", err)
		return false
	}

	file, ok := fs.GetFile(Filename(ccl.Filename))
	if !ok {
		return false
	}
	if file.Creator == ccl.Creator {
		ccl.NotifyChannel <- 1
		return true
	}
	return false
}

func (ccl CreateConfirmationListener) IsExpired() bool {
	return isPastTime(ccl.ExpirationTime)
}

type DeleteConfirmationListener struct {
	Filename string
	MinerState MinerState
	ConfirmsPerFileAppend int
	ConfirmsPerFileCreate int
	NotifyChannel chan int
	ExpirationTime time.Time
}

func (dcl DeleteConfirmationListener) TreeEventHandler() bool {
	fs, err := dcl.MinerState.GetFilesystemState(
		dcl.ConfirmsPerFileCreate,
		dcl.ConfirmsPerFileAppend)
	if err != nil {
		lg.Println("DeleteConfirmationListener, ", err)
		return false
	}

	_, exists := fs.GetFile(Filename(dcl.Filename))
	if !exists {
		dcl.NotifyChannel <- 1
	}
	return !exists
}

func (dcl DeleteConfirmationListener) IsExpired() bool {
	return isPastTime(dcl.ExpirationTime)
}

// Helpers
func isPastTime(expirationTime time.Time) bool {
	return time.Now().After(expirationTime)
}
