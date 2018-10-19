package api

import (
	"../../crypto"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

type fakeState struct {
}

func (fakeState) GetNode(id string) (*crypto.Block, bool) {
	return &bk, true
}

func (fakeState) GetRoots() []*crypto.Block {
	return []*crypto.Block{&bk}
}

func (fakeState) AddBlock(b *crypto.Block) {
	if !reflect.DeepEqual(bk, *b) {
		panic("error, blocks weren't equal")
	}
}

func (fakeState) AddJob(b *crypto.BlockOp) {
	if !reflect.DeepEqual(bkJob, *b) {
		panic("error, blocks weren't equal")
	}
}

var bkJob = crypto.BlockOp{
	Type: crypto.CreateFile,
	Creator: "me",
	Data: [crypto.DataBlockSize]byte{},
	Filename: "potato",
	RecordNumber: 3,
}

var bk = crypto.Block{
	MinerId: "1",
	Nonce: 2,
	PrevBlock: [16]byte{},
	Records: []*crypto.BlockOp{
		{
			Type: crypto.CreateFile,
		},
	},
	Type: crypto.RegularBlock,
}

var state = fakeState{}

var host = ":1222"

func init(){
	var e = InitMinerServer(host, state)
	if e != nil {
		panic("couldnt init server")
	}
}


func TestGetNodeTest(t *testing.T) {
	c, err := NewMinerCliet("localhost" + host)

	if err != nil {
		t.Fail()
	}

	nd, ok, _ := c.GetNode("a")
	if !ok {
		t.Fail()
	}
	equals(t, bk, *nd)

}

func TestAddNodeTest(t *testing.T) {
	c, err := NewMinerCliet("localhost" + host)

	if err != nil {
		t.Fail()
	}

	c.SendBlock(&bk)
}

func TestGetRoots(t *testing.T) {
	c, err := NewMinerCliet("localhost" + host)

	if err != nil {
		t.Fail()
	}

	rt, _ := c.GetRoots()
	equals(t, bk, *rt[0])
}

func TestSendJob(t *testing.T) {
	c, err := NewMinerCliet("localhost" + host)

	if err != nil {
		t.Fail()
	}

	c.SendJob(&bkJob)
}


// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
