package main

import (
	"../crypto"
	"../fdlib"
	. "../shared"
	"./state"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

// Miner type declaration
type Miner interface {
	CreateFileHandler(fname string) (errorType FailureType)
	// ListFilesHandler() does not return any error because on the client-side, the only kind of error
	// that can be returned is a DisconnectedError, which means we would never reach the Miner in the first place.
	ListFilesHandler() (fnames []string)
	TotalRecsHandler(fname string) (numRecs uint16, errorType FailureType)
	ReadRecHandler(fname string, recordNum uint16) (record [512]byte, errorType FailureType)
	AppendRecHandler(fname string, record [512]byte) (recordNum uint16, errorType FailureType)
}

type MinerConfiguration struct {
	MinedCoinsPerOpBlock uint8
	MinedCoinsPerNoOpBlock uint8
	NumCoinsPerFileCreate uint8
	GenOpBlockTimeout uint8
	GenesisBlockHash string
	PowPerOpBlock uint8
	PowPerNoOpBlock uint8
	ConfirmsPerFileCreate uint8
	ConfirmsPerFileAppend uint8
	MinerID string
	PeerMinersAddrs []string
	IncomingMinersAddr string
	OutgoingMinersIP string
	IncomingClientsAddr string
}

var lg = log.New(os.Stdout, "miner: ", log.Ltime)

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		lg.Println("usage: go run miner.go [settings]")
		os.Exit(1)
	}

	// Parse configuration
	conf, err := ParseConfig(argsWithoutProg[0])
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	}

	// Initialize miner state
	var blockHashBytes [md5.Size]byte
	blockHashBytesFull, err := hex.DecodeString(conf.GenesisBlockHash)
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	}
	copy(blockHashBytes[:], blockHashBytesFull[:md5.Size])

	minerStateConf := state.Config{
		AppendFee: state.Balance(1),
		CreateFee: state.Balance(conf.NumCoinsPerFileCreate),
		OpReward: state.Balance(conf.MinedCoinsPerOpBlock),
		NoOpReward: state.Balance(conf.MinedCoinsPerNoOpBlock),
		OpNumberOfZeros: int(conf.PowPerOpBlock),
		NoOpNumberOfZeros: int(conf.PowPerNoOpBlock),
		Address: conf.IncomingMinersAddr, // todo ksenia. we have several addresses in config, need to update this
		ConfirmsPerFileCreate: int(conf.ConfirmsPerFileCreate),
		ConfirmsPerFileAppend: int(conf.ConfirmsPerFileAppend),
		OpPerBlock: 30, // todo victor can you give me some insight as to what this value should be?
		MinerId: conf.MinerID,
		GenesisBlockHash: blockHashBytes,
		GenOpBlockTimeout: conf.GenOpBlockTimeout,
	}
	ms := state.NewMinerState(minerStateConf, conf.PeerMinersAddrs)

	// Initialize miner instance
	var minerInstance Miner = MinerInstance{minerConf: conf, minerState: ms}

	// Initialize failure detector and start responding
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	epochNonce := r1.Uint64()
	fd, _, err := fdlib.InitializeFDLib(uint64(epochNonce), 5)
	if err != nil {
		lg.Println(err)
		os.Exit(1)
	}
	fd.StartResponding(conf.IncomingClientsAddr)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	lg.Println("Listening for clients...")

	ci := ClientHandler{
		ListenHost: conf.IncomingClientsAddr,
		miner:      &minerInstance,
		waitGroup:  wg,
	}
	go ci.ListenForClients()
	wg.Wait()
}

type MinerInstance struct {
	minerConf MinerConfiguration
	minerState state.MinerState
}

// errorType can be one of: FILE_EXISTS, BAD_FILENAME, NO_ERROR
func (miner MinerInstance) CreateFileHandler(fname string) (errorType FailureType) {
	for {
		lg.Println("Handling create file request")

		// create job
		job := new(crypto.BlockOp)
		job.Type = crypto.CreateFile
		job.Creator = miner.minerConf.MinerID
		job.Filename = fname

		// validate against file system, accounts states
		_, acctsErr, filesErr := miner.minerState.ValidateJobSet([]*crypto.BlockOp{job})

		if acctsErr != nil {
			singleAcctsErr := getSingleAccountsError(acctsErr)
			if singleAcctsErr == NOT_ENOUGH_MONEY {
				time.Sleep(time.Second)
				continue
			}
		}

		if filesErr != nil {
			singleFilesErr := getSingleFilesError(filesErr)
			if singleFilesErr == FILE_EXISTS || singleFilesErr == BAD_FILENAME {
				return singleFilesErr
			}
		}

		// add job wait for it to complete
		miner.minerState.AddJob(*job)
		ccl := state.CreateConfirmationListener {
			Creator: miner.minerConf.MinerID,
			Filename: fname,
			MinerState: miner.minerState,
			ConfirmsPerFileAppend: int(miner.minerConf.ConfirmsPerFileAppend),
			ConfirmsPerFileCreate: int(miner.minerConf.ConfirmsPerFileCreate),
			NotifyChannel: make(chan int, 100),
		}
		miner.minerState.AddTreeListener(ccl)
		for {
			select {
			case <- ccl.NotifyChannel:
				return NO_ERROR
			default:
				// do nothing
			}
		}
	}
}

func (miner MinerInstance) ListFilesHandler() (fnames []string) {
	lg.Println("Handling list files request")

	fs := miner.getFileSystemState()

	files := fs.GetAll()
	fnames = make([]string, len(files))
	i := 0
	for key := range files {
		fnames[i] = string(key)
		i++
	}
	return fnames
}

// errorType can be one of: FILE_DOES_NOT_EXIST, NO_ERROR
func (miner MinerInstance) TotalRecsHandler(fname string) (numRecs uint16, errorType FailureType) {
	lg.Println("Handling total records request")

	fs := miner.getFileSystemState()

	file, ok := fs.GetFile(Filename(fname))
	if !ok {
		return 0, FILE_DOES_NOT_EXIST
	}
	return file.NumberOfRecords, NO_ERROR
}

// errorType can be one of: FILE_DOES_NOT_EXIST, NO_ERROR
func (miner MinerInstance) ReadRecHandler(fname string, recordNum uint16) (record [512]byte, errorType FailureType) {
	lg.Println("Handling read record request")
	var read_result [512]byte

	fs := miner.getFileSystemState()

	file, ok := fs.GetFile(Filename(fname))
	if !ok {
		return read_result, FILE_DOES_NOT_EXIST
	}

	offset := recordNum * 512
	copy(read_result[:], file.Data[offset:offset+512])
	return read_result, NO_ERROR
}

// errorType can be one of: FILE_DOES_NOT_EXIST, MAX_LEN_REACHED, NO_ERROR
func (miner MinerInstance) AppendRecHandler(fname string, record [512]byte) (recordNum uint16, errorType FailureType) {
	for {
		lg.Println("Handling append record request")

		fs := miner.getFileSystemState()

		// check if file already exists
		file, ok := fs.GetFile(Filename(fname))
		if !ok {
			return 0, FILE_DOES_NOT_EXIST
		}

		// create job
		job := new(crypto.BlockOp)
		job.Type = crypto.AppendFile
		job.Creator = miner.minerConf.MinerID
		job.Filename = fname
		job.RecordNumber = file.NumberOfRecords
		copy(job.Data[:], record[:])

		// validate against file system, accounts states
		_, acctsErr, filesErr := miner.minerState.ValidateJobSet([]*crypto.BlockOp{job})

		if acctsErr != nil {
			singleAcctsErr := getSingleAccountsError(acctsErr)
			if singleAcctsErr == NOT_ENOUGH_MONEY {
				time.Sleep(time.Second)
				continue
			}
		}

		if filesErr != nil {
			singleFilesErr := getSingleFilesError(filesErr)
			if singleFilesErr == FILE_DOES_NOT_EXIST || singleFilesErr == MAX_LEN_REACHED {
				return 0, singleFilesErr
			} else if singleFilesErr == APPEND_DUPLICATE {
				continue
			}
		}

		// add job wait for it to complete
		miner.minerState.AddJob(*job)
		acl := state.AppendConfirmationListener {
			Creator: miner.minerConf.MinerID,
			Filename: fname,
			RecordNumber: job.RecordNumber,
			Data: record,
			MinerState: miner.minerState,
			ConfirmsPerFileAppend: int(miner.minerConf.ConfirmsPerFileAppend),
			ConfirmsPerFileCreate: int(miner.minerConf.ConfirmsPerFileCreate),
			NotifyChannel: make(chan int, 100),
		}
		miner.minerState.AddTreeListener(acl)
		for {
			select {
			case <- acl.NotifyChannel:
				return 0, NO_ERROR
			default:
				// do nothing
			}
		}
	}
}

/////////// Helpers ///////////////////////////////////////////////////////

func ParseConfig(fileName string) (MinerConfiguration, error){
	var m MinerConfiguration

	file, err := os.Open(fileName)
	if err != nil {
		return MinerConfiguration{}, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return MinerConfiguration{}, err
	}

	err = json.Unmarshal(b, &m)
	if err != nil {
		return MinerConfiguration{}, err
	}
	return m, nil
}

func (miner MinerInstance) getFileSystemState() state.FilesystemState {
	fs, err := miner.minerState.GetFilesystemState(
		int(miner.minerConf.ConfirmsPerFileCreate),
		int(miner.minerConf.ConfirmsPerFileAppend))
	if err != nil {
		// todo ksenia what to do about this case?
		panic(err)
	}
	return fs
}

func getSingleFilesError(compositeError error) FailureType {
	// todo ksenia make this better
	if cerr, ok := compositeError.(state.CompositeError); ok {
		if _, ok := cerr.Current.(state.FileAlreadyExistsValidationError); ok {
			return FILE_EXISTS
		} else if _, ok := cerr.Current.(state.BadFileNameValidationError); ok {
			return BAD_FILENAME
		} else if _, ok := cerr.Current.(state.FileDoesNotExistValidationError); ok {
			return FILE_DOES_NOT_EXIST
		} else if _, ok := cerr.Current.(state.AppendDuplicateValidationError); ok {
			return APPEND_DUPLICATE
		} else if _, ok := cerr.Current.(state.MaxLengthReachedValidationError); ok {
			return MAX_LEN_REACHED
		}
	}
	return NO_ERROR
}

func getSingleAccountsError(compositeError error) FailureType {
	// todo ksenia make this better
	if cerr, ok := compositeError.(state.CompositeError); ok {
		if _, ok := cerr.Current.(state.NotEnoughMoneyValidationError); ok {
			time.Sleep(time.Second)
			return NOT_ENOUGH_MONEY
		}
	}
	return NO_ERROR
}
