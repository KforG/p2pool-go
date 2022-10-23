package main

import (
	"os"
	"time"

	"github.com/gertjaap/p2pool-go/config"
	"github.com/gertjaap/p2pool-go/logging"
	p2pnet "github.com/gertjaap/p2pool-go/net"
	"github.com/gertjaap/p2pool-go/p2p"
	"github.com/gertjaap/p2pool-go/work"
)

func main() {
	logging.SetLogLevel(int(logging.LogLevelDebug))
	logFile, _ := os.OpenFile("p2pool.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	logging.SetLogFile(logFile)

	config.LoadConfig()

	p2pnet.ActiveNetwork = p2pnet.Vertcoin()

	sc := work.NewShareChain()
	err := sc.Load()
	if err != nil {
		panic(err)
	}

	//return
	pm := p2p.NewPeerManager(p2pnet.ActiveNetwork, sc)

	go func() {
		for s := range sc.NeedShareChannel {
			pm.AskForShare(s)
		}
	}()

	for {
		logging.Debugf("Number of active peers: %d", pm.GetPeerCount())
		time.Sleep(time.Second * 5)
	}
}
