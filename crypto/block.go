package crypto

import (
	"../shared/datastruct"
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"math/rand"
	"unsafe"
)

const DataBlockSize = 512

type BlockOpType uint32
type BlockOpData [DataBlockSize]byte

const (
	CreateFile BlockOpType = iota
	AppendFile
	DeleteFile
)

type BlockOp struct {
	Type BlockOpType
	// Miner id of the person that create the request
	Creator string
	Filename string
	RecordNumber uint16
	Data BlockOpData
}

type BlockType int

const (
	NoOpBlock BlockType = iota
	RegularBlock
	GenesisBlock
)

type Block struct {
	Type BlockType

	// In the case of any regular block this holds the hash of the preceding node
	// however if the block is of type GenesisBlock, it will hold that block id
	PrevBlock [md5.Size]byte
	Records   []*BlockOp
	MinerId   string
	Nonce     uint32
}

func (b *Block) serialize() []byte {
	buf := &bytes.Buffer{}
	buf.Write(b.PrevBlock[:])

	intBuff := make([]byte, unsafe.Sizeof(uint32(1)))
	for _, v := range b.Records {
		buf.Write([]byte(v.Filename))
		buf.Write(v.Data[:])
		binary.LittleEndian.PutUint16(intBuff, v.RecordNumber)
		buf.Write(intBuff)
	}

	buf.Write([]byte(b.MinerId))
	binary.LittleEndian.PutUint32(intBuff, b.Nonce)
	buf.Write(intBuff)
	return buf.Bytes()
}

func (b *Block) hash(ser []byte) []byte {
	switch b.Type {
	case NoOpBlock, RegularBlock:
		sum := md5.Sum(ser)
		return sum[:]
	case GenesisBlock:
		return b.PrevBlock[:]
	}

	panic("cannot hash block")
}

func (b *Block) Hash() []byte {
	return b.hash(b.serialize())
}

func (b *Block) Id() string {
	return fmt.Sprintf("%x", b.Hash())
}

func (b *Block) valid(ser []byte, zeros int) bool {
	hash := b.hash(ser)
	for i := len(hash) - 1; i >= 0 && zeros > 0; zeros, i = zeros-8, i-1 {
		mask := uint8(0xFF)
		if zeros < 8 {
			mask = mask >> uint(7-zeros)
		}
		if hash[i]&mask != 0 {
			return false
		}
	}
	return true
}

func (b *Block) GetZerosForType(zerosOp int, zerosNoOp int) int {
	var zeros int
	switch b.Type {
	case GenesisBlock, RegularBlock:
		zeros = zerosOp
	case NoOpBlock:
		zeros = zerosNoOp
	}
	return zeros
}

func (b *Block) Valid(zerosOp int, zerosNoOp int) bool {
	zeros := b.GetZerosForType(zerosOp, zerosNoOp)
	return b.valid(b.serialize(), zeros)
}

func (b *Block) FindNonce(zerosOp int, zerosNoOp int) {
	zeros := b.GetZerosForType(zerosOp, zerosNoOp)
	b.FindNonceWithStopSignal(zeros, new(bool))
}

func (b *Block) FindNonceWithStopSignal(zeros int, stopSig *bool) {
	start := uint32(rand.Int())
	ser := b.serialize()

	for !b.valid(ser, zeros) && !*stopSig {
		b.Nonce = start

		intBuff := make([]byte, unsafe.Sizeof(uint32(1)))
		binary.LittleEndian.PutUint32(intBuff, b.Nonce)
		copy(ser[len(ser)-4:], intBuff)

		start += 1
	}
}

type BlockElement struct {
	Block *Block
}

func (b BlockElement) Encode() []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(b.Block)
	if err != nil {
		log.Printf("Couldn't encode block: %v\n", b.Block)
	}
	return buf.Bytes()
}

func (b BlockElement) New(r io.Reader) datastruct.Element {
	newBlock := Block{}
	newBe := BlockElement{
		Block: &newBlock,
	}
	dec := gob.NewDecoder(r)
	err := dec.Decode(&newBlock)

	if err != nil {
		log.Printf("Couldn't decode a block: %v", err)
		return nil
	}

	return newBe
}

func (b BlockElement) ParentId() string {
	return fmt.Sprintf("%x", b.Block.PrevBlock[:])
}

func (b BlockElement) Id() string {
	return b.Block.Id()
}
