package rpc

import (
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/gertjaap/p2pool-go/net"
)

func GetBlockTemplate() (*btcjson.GetBlockTemplateResult, error) {
	req := btcjson.TemplateRequest{
		Mode: "template",
	}
	if net.ActiveNetwork.Softforks != nil {
		req.Rules = net.ActiveNetwork.Softforks
	}

	blockTemplateResult, err := ConnRPC.GetBlockTemplate(&req)
	if err != nil {
		return blockTemplateResult, err
	}

	return blockTemplateResult, nil
}

func SubmitBlock(block *btcutil.Block, options *btcjson.SubmitBlockOptions) error {
	err := ConnRPC.SubmitBlock(block, options)
	if err != nil {
		// Retry one time
		time.Sleep(1 * time.Second)
		err = ConnRPC.SubmitBlock(block, options)
		return err
	}
	return nil
}
