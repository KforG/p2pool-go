package net

import (
	"encoding/hex"

	"github.com/gertjaap/p2pool-go/logging"
	verthash "github.com/gertjaap/verthash-go"
)

func Vertcoin(testnet bool) Network {
	if testnet {
		return TestnetVertcoin()
	}

	n := Network{P2PPort: 9346}
	n.RPCPort = 5888
	n.WorkerPort = 9171
	n.MessagePrefix, _ = hex.DecodeString("7c3614a6bcdcf784")
	n.Identifier, _ = hex.DecodeString("a06a81c827cab983")
	n.ChainLength = 5100
	n.SeedHosts = []string{"localhost", "p2proxy.vertcoin.org", "mindcraftblocks.com", "vtc-fl.javerity.com"}
	n.Softforks = []string{"bip34", "bip66", "bip65", "csv", "segwit", "taproot"}
	n.POWHash = func(b []byte) []byte {
		vh, err := verthash.NewVerthash("verthash.dat", true)
		if err != nil {
			logging.Errorf("Failed to load verthash.dat into memory: %v", err)
			panic(err)
		}
		defer vh.Close()
		res, _ := vh.SumVerthash(b)
		return res[:]
	}

	// Verify verthash.dat is present and okay
	var ok bool = false
	for !ok {
		logging.Infof("Verifying verthash.dat... This can take a few moments\n")
		ok, err := verthash.VerifyVerthashDatafile("verthash.dat")
		if ok {
			return n
		}
		logging.Errorf("Datafile failed verification: %v\n", err)
		logging.Infof("Creating new datafile... This can take a while\n")
		verthash.EnsureVerthashDatafile("verthash.dat")
	}

	return n
}
