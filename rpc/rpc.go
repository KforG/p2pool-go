package rpc

import (
	"fmt"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gertjaap/p2pool-go/config"
	"github.com/gertjaap/p2pool-go/logging"
	"github.com/gertjaap/p2pool-go/net"
)

var ConnRPC *rpcclient.Client

func InitRPC() error {
	connCfg := &rpcclient.ConnConfig{
		Host:         fmt.Sprintf("127.0.0.1:%d", net.ActiveNetwork.RPCPort),
		User:         config.Active.RPCUser,
		Pass:         config.Active.RPCPass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	logging.Infof("Connecting to RPC server with user: %s\n", connCfg.User)
	conn, err := rpcclient.New(connCfg, nil)
	if err != nil {
		logging.Errorf("Failed to connect to RPC server..\n", err)
		return err
	}
	logging.Infof("Connection to the RPC server established successfully\n")
	ConnRPC = conn
	return nil
}
