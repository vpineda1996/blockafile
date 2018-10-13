package crypto

import (
	"bytes"
	"encoding/gob"
	"io"
	"../shared/mrootedtree"
	"log"
)

type BlockRecord [512]byte

type Block struct {
	PrevBlock string
	Records []BlockRecord
	MinerId string
	Nonce string
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

func ( BlockElement) New(r io.Reader) mrootedtree.Element {
	newB := BlockElement{}
	dec := gob.NewDecoder(r)
	err := dec.Decode(&newB)

	if err != nil {
		log.Fatal("Couldn't decode a block")
		return nil
	}

	return newB
}

func (b BlockElement) Id() string {
	// todo figure out how to find the current hashId
	panic("implement me")
}
