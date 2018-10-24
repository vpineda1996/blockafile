package main

import (
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		_, err := ParseConfig("../testfiles/config_nope.json")
		assert(t, err != nil, "should fail on parsing non-existent file")
	})

	t.Run("badly-formed file", func(t *testing.T) {
		_, err := ParseConfig("../testfiles/config_bad.json")
		assert(t, err != nil, "should fail on parsing badly-formed file")
	})

	t.Run("well-formed file with fields missing", func(t *testing.T) {
		_, err := ParseConfig("../testfiles/config_missing.json")
		assert(t, err == nil, "should parse well-formed config file with some fields renamed/missing")
	})

	t.Run("well-formed file with all fields", func(t *testing.T) {
		mc, err := ParseConfig("../testfiles/config_good.json")
		assert(t, err == nil, "should parse well-formed config file")
		equals(t, uint8(8), mc.MinedCoinsPerOpBlock)
		equals(t, uint8(4), mc.MinedCoinsPerNoOpBlock)
		equals(t, uint8(4), mc.NumCoinsPerFileCreate)
		equals(t, uint8(5), mc.GenOpBlockTimeout)
		equals(t, "83218ac34c1834c26781fe4bde918ee4", mc.GenesisBlockHash)
		equals(t, uint8(4), mc.PowPerOpBlock)
		equals(t, uint8(4), mc.PowPerNoOpBlock)
		equals(t, uint8(2), mc.ConfirmsPerFileCreate)
		equals(t, uint8(4), mc.ConfirmsPerFileAppend)
		equals(t, "Mijnwerker", mc.MinerID)
		equals(t, []string{"127.0.0.1:5050", "127.0.0.1:6060", "127.0.0.1:7070"}, mc.PeerMinersAddrs)
		equals(t, "127.0.0.1:8080", mc.IncomingMinersAddr)
		equals(t, "127.0.0.1", mc.OutgoingMinersIP)
		equals(t, "127.0.0.1:9090", mc.IncomingClientsAddr)
	})
}
