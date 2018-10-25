package integration_tests

import (
	"../crypto"
	. "../miner/state"
	"../shared"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

var config = Config{
	GenesisBlockHash:      [md5.Size]byte{1, 2, 3, 4, 5},
	NumberOfZeros:         16,
	MinerId:               "1",
	Address:               "localhost:8085",
	AppendFee:             1,
	ConfirmsPerFileAppend: 3,
	ConfirmsPerFileCreate: 5,
	CreateFee:             2,
	NoOpReward:            1,
	OpPerBlock:            3,
	OpReward:              2,
	GenOpBlockTimeout:     100,
}

var connectingNodes = []string{}

func TestMinerState(t *testing.T) {
	// Start a miner state
	s := NewMinerState(config, connectingNodes)

	// wait for some no-op blocks to be generated
	time.Sleep(time.Second)

	// ----------------------------------------------------
	// no-op mining thread is running
	roots := s.GetRoots()
	equals(t, 1, len(roots))
	assert(t, !reflect.DeepEqual(config.GenesisBlockHash[:], roots[0].Hash()), "should not be genesis block")
	equals(t, crypto.NoOpBlock, roots[0].Type)
	equals(t, config.MinerId, roots[0].MinerId)

	// ----------------------------------------------------
	// create new job
	job := crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile",
		RecordNumber: 0,
	}
	s.AddJob(job)
	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err := s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, shared.FileData{}, fs.GetAll()["myFile"].Data)

	// ----------------------------------------------
	// process two jobs at a time
	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile",
		RecordNumber: 0,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile",
		RecordNumber: 1,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint32(2), fs.GetAll()["myFile"].NumberOfRecords)

	// ----------------------------------------------
	// handles jobs that are flawed -- create file
	job = crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile",
		RecordNumber: 1,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint32(2), fs.GetAll()["myFile"].NumberOfRecords)

	// ----------------------------------------------
	// handles jobs that are flawed -- append
	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{1, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile",
		RecordNumber: 1,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint32(2), fs.GetAll()["myFile"].NumberOfRecords)

	// -----------------------------------------------
	// respects order of ops inside block -- create and append

	job = crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint32(1), fs.GetAll()["myFile2"].NumberOfRecords)

	// -----------------------------------------------
	// respects order of ops inside block -- create and 3x append and ignore second creation

	job = crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 1,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 2,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint32(3), fs.GetAll()["myFile2"].NumberOfRecords)

	// -----------------------------------------------
	// money validation fixes errors and deletes necessary

	job = crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{},
		Creator:      "alpha master",
		Filename:     "myFile3",
		RecordNumber: 0,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile3",
		RecordNumber: 0,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile3",
		RecordNumber: 1,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile3",
		RecordNumber: 2,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	_, v := fs.GetAll()["myFile3"]
	equals(t, false, v)

	// -----------------------------------------------
	// delete a file
	job = crypto.BlockOp{
		Type:         crypto.DeleteFile,
		Data:         [crypto.DataBlockSize]byte{},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	_, v = fs.GetAll()["myFile2"]
	equals(t, false, v)

	// -----------------------------------------------
	// create the a new file, with the same name

	job = crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      config.MinerId,
		Filename:     "myFile2",
		RecordNumber: 0,
	}
	s.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = s.GetFilesystemState(config.ConfirmsPerFileCreate, config.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint32(1), fs.GetAll()["myFile2"].NumberOfRecords)

}

// Taken from https://github.com/benbjohnson/testing
// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d: unexpected error: %str\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}
