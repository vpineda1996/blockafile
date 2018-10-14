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
	"unsafe"
)

const DataBlockSize = 512

type BlockOpType uint32
type BlockOpData [DataBlockSize]byte
const (
	CreateFile BlockOpType = iota
	AppendFile
)
type BlockOp struct {
	Type BlockOpType
	// Miner id of the person that create the request
	Creator string
	Filename string
	RecordNumber uint32
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
	Records []*BlockOp
	MinerId string
	Nonce uint32
}

func (b *Block) Hash() []byte {
	// create a buffer to create a sum
	switch b.Type {
	case NoOpBlock, RegularBlock:
		buf := &bytes.Buffer{}
		buf.Write(b.PrevBlock[:])

		intBuff := make([]byte, unsafe.Sizeof(uint32(1)))
		for _, v := range b.Records {
			buf.Write([]byte(v.Filename))
			buf.Write(v.Data[:])
			binary.LittleEndian.PutUint32(intBuff, v.RecordNumber)
			buf.Write(intBuff)
		}

		buf.Write([]byte(b.MinerId))
		binary.LittleEndian.PutUint32(intBuff, b.Nonce)
		buf.Write(intBuff)

		sum := md5.Sum(buf.Bytes())
		return sum[:]
	case GenesisBlock:
		return b.PrevBlock[:]
	}
	panic("cannot hash block")
}


type BlockElement struct {
	Block *Block
}

func (b BlockElement) Encode() []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(b.Block)
	if err != nil {
		log.Fatalf("Couldn't encode block: %v", b.Block)
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
		log.Fatalf("Couldn't decode a block: %v", err)
		return nil
	}

	return newBe
}

func (b BlockElement) ParentId() string {
	return fmt.Sprintf("%x", b.Block.PrevBlock[:])
}

func (b BlockElement) Id() string {
	return fmt.Sprintf("%x", b.Block.Hash())
}
