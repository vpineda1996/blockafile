package rfslib_integration_test

import (
	"../../fdlib"
	"../../miner/instance"
	"../../rfslib"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestRFSLibMinerIntegration(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	instance.NewMinerInstance("./testfiles/config_nopeers.json", wg, false)
	time.Sleep(time.Second)

	fdlib.IgnoreInstanceCheck = true

	// --------------------------------
	// Make some requests to the miner

	const (
		SAMPLE_FNAME = "sample_file1"
		localAddr = "localhost:5152"
		minerAddr = "localhost:9091"
	)

	rfs, err := rfslib.Initialize(localAddr, minerAddr)
	ok(t, err)

	// Create file
	err = rfs.CreateFile(SAMPLE_FNAME)
	ok(t, err)

	// Append record
	recordContents := []byte("new record")
	record := new(rfslib.Record)
	copy(record[:], recordContents)
	recNum, err := rfs.AppendRec(SAMPLE_FNAME, record)
	ok(t, err)
	equals(t, uint16(0), recNum)

	// List files
	fnames, err := rfs.ListFiles()
	ok(t, err)
	equals(t, 1, len(fnames))

	// Total records count
	numRecs, err := rfs.TotalRecs(SAMPLE_FNAME)
	ok(t, err)
	equals(t, uint16(1), numRecs)

	// Read record
	record = new(rfslib.Record)
	index := uint16(0)
	err = rfs.ReadRec(SAMPLE_FNAME, index, record)
	ok(t, err)
	equals(t, recordContents, record[:len(recordContents)])

	// delete record
	err = rfs.DeleteFile(SAMPLE_FNAME)
	ok(t, err)

	// check file got deleted
	fnames, err = rfs.ListFiles()
	ok(t, err)
	equals(t, 0, len(fnames))
}

/****************** Comment this test out if you don't want to wait forever ******************/
/*func TestAppendUntilFileLimit(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	instance.NewMinerInstance("./testfiles/config_nopeers.json", wg)
	time.Sleep(time.Second)

	fdlib.IgnoreInstanceCheck = true

	// --------------------------------
	// Make some requests to the miner

	const (
		SAMPLE_FNAME = "sample_file1"
		localAddr = "localhost:5152"
		minerAddr = "localhost:9091"
	)

	rfs, err := rfslib.Initialize(localAddr, minerAddr)
	ok(t, err)

	// Create file
	err = rfs.CreateFile(SAMPLE_FNAME)
	ok(t, err)

	// Append file until we reach the file limit
	for i := 0; i < 65535; i++ {
		recordContents := []byte("new record")
		record := new(rfslib.Record)
		copy(record[:], recordContents)
		recNum, err := rfs.AppendRec(SAMPLE_FNAME, record)
		ok(t, err)
		equals(t, uint16(i), recNum)
	}

	// Append one last time, make sure it fails
	recordContents := []byte("new record")
	record := new(rfslib.Record)
	copy(record[:], recordContents)
	_, err = rfs.AppendRec(SAMPLE_FNAME, record)
	assert(t, err != nil, "should have returned an error")
	ok(t, err)
}*/

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



