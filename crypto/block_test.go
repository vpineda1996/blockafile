package crypto

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestSimpleBlock(t *testing.T) {
	record := BlockOp{
		Type:CreateFile,
		Creator: "",
		Data: BlockOpData{20},
		Filename: "",
	}
	records := make([]*BlockOp, 1)
	records[0] = &record
	prevBlock := [md5.Size]byte {20, 32, 1}
	minerId := "asdasf122"
	nonce := uint32(232412)
	t.Run("simple hashing for a no block", func(t *testing.T) {
		bk := Block{
			Type: NoOpBlock,
			Nonce: nonce,
			MinerId: minerId,
			PrevBlock: prevBlock,
			Records: make([]*BlockOp, 0),
		}
		equals(t, []byte{86, 84, 86 ,126 ,15 ,25, 91, 19 ,255, 232, 161, 94, 5 ,164, 249, 15},bk.Hash())
	})

	t.Run("simple for a genesis block", func(t *testing.T) {
		bk := Block{
			Type: GenesisBlock,
			Nonce: nonce,
			MinerId: minerId,
			PrevBlock: prevBlock,
			Records: records,
		}
		equals(t, []byte{20,32, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},bk.Hash())
	})

	t.Run("hashing for a regular block with one entry", func(t *testing.T) {
		bk := Block{
			Type: RegularBlock,
			Nonce: nonce,
			MinerId: minerId,
			PrevBlock: prevBlock,
			Records: records,
		}
		equals(t,
			[]byte{0xa4, 0xef, 0xe6, 0xb2, 0x97, 0x3b, 0xeb, 0x25, 0x4e, 0xed, 0x81, 0x63, 0x14, 0x5f, 0xaf, 0xda},
			bk.Hash())
	})
}

func TestEncoding(t *testing.T) {
	record := BlockOp{
		Type:CreateFile,
		Creator: "",
		Data: BlockOpData{20},
		Filename: "",
	}
	records := make([]*BlockOp, 1)
	records[0] = &record
	prevBlock := [md5.Size]byte {20, 32, 1}
	minerId := "asdasf122"
	nonce := uint32(232412)

	t.Run("hashing for a regular block with one entry", func(t *testing.T) {
		bk := Block{
			Type: RegularBlock,
			Nonce: nonce,
			MinerId: minerId,
			PrevBlock: prevBlock,
			Records: records,
		}

		be := BlockElement{
			Block: &bk,
		}
		enc := be.Encode()
		btck := be.New(bytes.NewReader(enc))

		equals(t, be.Id(), btck.Id())
	})
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