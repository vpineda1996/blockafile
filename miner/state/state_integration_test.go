package state

import (
	"crypto/md5"
	"testing"
)

var config = Config{
	genesisBlockHash: [md5.Size]byte{1, 2, 3, 4, 5},
	numberOfZeros: 20,
	minerId: "1",
	address: ":8080",
	appendFee: 1,
	confirmsPerFileAppend: 3,
	confirmsPerFileCreate: 5,
	createFee: 2,
	noOpReward: 1,
	opPerBlock: 3,
	opReward: 2,
	GenOpBlockTimeout: 100,
}

var connectingNodes = []string{}

func TestAFullMinerStateWithoutOtherNeighbourMiners(t *testing.T) {
	// Start a miner state
	s := NewMinerState(config, connectingNodes)

	// genesis block should be there
	roots := s.GetRoots()
	equals(t, 1, len(roots))
	equals(t, config.genesisBlockHash[:], roots[0].Hash())


}
