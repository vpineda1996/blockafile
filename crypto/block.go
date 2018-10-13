package crypto

import (
	"../shared/tree"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
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
	Filename string
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

		for _, v := range b.Records {
			buf.Write([]byte(v.Filename))
			buf.Write(v.Data[:])
		}

		buf.Write([]byte(b.MinerId))

		nonceEnc := make([]byte, unsafe.Sizeof(uint32(1)))
		binary.LittleEndian.PutUint32(nonceEnc, b.Nonce)
		buf.Write(nonceEnc)

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

func (b BlockElement) New(r io.Reader) tree.Element {
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

func (b BlockElement) Id() string {
	return base64.StdEncoding.EncodeToString(b.Block.Hash())
}
