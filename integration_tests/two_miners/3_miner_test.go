package two_miners

import (
	"../../crypto"
	. "../../miner/state"
	"../../shared"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

const BobAddress = "localhost:8080"
const AliceAddress = "localhost:8081"
const ClaudiaAddress = "localhost:8082"
var GenesisBlockHash = [md5.Size]byte{1, 2, 3, 4, 5}

var dificulty = 18

var BobConfig = Config{
	GenesisBlockHash:      GenesisBlockHash,
	OpNumberOfZeros:       dificulty,
	NoOpNumberOfZeros:     dificulty,
	MinerId:               "bob",
	Address:               BobAddress,
	AppendFee:             1,
	ConfirmsPerFileAppend: 5,
	ConfirmsPerFileCreate: 5,
	CreateFee:             2,
	NoOpReward:            1,
	OpPerBlock:            3,
	OpReward:              2,
	GenOpBlockTimeout:     100,
}

var ClaudiaConfig = Config{
	GenesisBlockHash:      GenesisBlockHash,
	OpNumberOfZeros:       dificulty,
	NoOpNumberOfZeros:     dificulty,
	MinerId:               "claudia",
	Address:               ClaudiaAddress,
	AppendFee:             1,
	ConfirmsPerFileAppend: 5,
	ConfirmsPerFileCreate: 5,
	CreateFee:             2,
	NoOpReward:            1,
	OpPerBlock:            3,
	OpReward:              2,
	GenOpBlockTimeout:     100,
}

var AliceConfig = Config{
	GenesisBlockHash:      GenesisBlockHash,
	OpNumberOfZeros:       dificulty,
	NoOpNumberOfZeros:     dificulty,
	MinerId:               "alice",
	Address:               AliceAddress,
	AppendFee:             1,
	ConfirmsPerFileAppend: 5,
	ConfirmsPerFileCreate: 5,
	CreateFee:             2,
	NoOpReward:            1,
	OpPerBlock:            3,
	OpReward:              2,
	GenOpBlockTimeout:     100,
}

var aliceNodes = []string{BobAddress}
var bobNodes = []string{}
var claudiaNodes = []string{AliceAddress}

func TestTwoMiners(t *testing.T) {
	// Start a miner state
	BobMiner := NewMinerState(BobConfig, bobNodes)
	time.Sleep(time.Second)
	AliceMiner := NewMinerState(AliceConfig, aliceNodes)

	// wait for some no-op blocks to be generated
	time.Sleep(time.Second)

	// ----------------------------------------------------
	// create new job and send it to Alice
	job := crypto.BlockOp{
		Type:         crypto.CreateFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      BobConfig.MinerId,
		Filename:     "myFile",
		RecordNumber: 0,
	}
	BobMiner.AddJob(job)
	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err := BobMiner.GetFilesystemState(BobConfig.ConfirmsPerFileCreate, BobConfig.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, shared.FileData{}, fs.GetAll()["myFile"].Data)

	// wait for alice to request roots
	time.Sleep(time.Second * 5)
	// block should have reached alice
	fs, err = AliceMiner.GetFilesystemState(AliceConfig.ConfirmsPerFileCreate, AliceConfig.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, shared.FileData{}, fs.GetAll()["myFile"].Data)

	//
	// ----------------------------------------------
	// bob recognizes alice is working on something
	BobMiner.SleepMiner()
	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      BobConfig.MinerId,
		Filename:     "myFile",
		RecordNumber: 0,
	}
	AliceMiner.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = BobMiner.GetFilesystemState(BobConfig.ConfirmsPerFileCreate, BobConfig.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint16(1), fs.GetAll()["myFile"].NumberOfRecords)

	//
	// ----------------------------------------------
	// claudia joins the network
	BobMiner.SleepMiner()
	AliceMiner.SleepMiner()
	ClaudiaMiner := NewMinerState(ClaudiaConfig, claudiaNodes)

	// let claudia mine some blocks for the lols
	time.Sleep(time.Second)
	ClaudiaMiner.SleepMiner()

	// wait for claudia to catch up
	time.Sleep(time.Second * 10)
	fs, err = ClaudiaMiner.GetFilesystemState(BobConfig.ConfirmsPerFileCreate, BobConfig.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint16(1), fs.GetAll()["myFile"].NumberOfRecords)


	//
	// ----------------------------------------------
	// clauida publishes a job to the network, bob mines it and claudia receives it
	BobMiner.ActivateMiner()
	job = crypto.BlockOp{
		Type:         crypto.AppendFile,
		Data:         [crypto.DataBlockSize]byte{9, 8, 7, 6, 5, 4, 3},
		Creator:      BobConfig.MinerId,
		Filename:     "myFile",
		RecordNumber: 1,
	}
	ClaudiaMiner.AddJob(job)

	// wait for job to be processed
	time.Sleep(time.Second * 5)
	fs, err = BobMiner.GetFilesystemState(BobConfig.ConfirmsPerFileCreate, BobConfig.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint16(2), fs.GetAll()["myFile"].NumberOfRecords)

	fs, err = ClaudiaMiner.GetFilesystemState(ClaudiaConfig.ConfirmsPerFileCreate, ClaudiaConfig.ConfirmsPerFileAppend)
	ok(t, err)
	equals(t, uint16(2), fs.GetAll()["myFile"].NumberOfRecords)

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
