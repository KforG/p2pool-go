package config

import (
	"flag"
	"os"

	"github.com/gertjaap/p2pool-go/logging"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Network string `yaml:"network"`
	RPCUser string `yaml:"rpcUser"`
	RPCPass string `yaml:"rpcPass"`
}

var Active Config

func LoadConfig() {
	// Load config file first
	file, err := os.Open("config.yaml")
	if err != nil {
		logging.Warnf("No config.yaml file found.")
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&Active)
	if err != nil {
		logging.Errorf("Failed to decode config.yaml: %v", err)
	}

	// Override config file if started with flags
	net := flag.String("n", "", "Network")
	user := flag.String("u", "", "RPC Username")
	pass := flag.String("p", "", "RPC Password")
	flag.Parse()

	if *net != "" {
		Active.Network = *net
	}
	if *user != "" {
		Active.RPCUser = *user
	}
	if *pass != "" {
		Active.RPCPass = *pass
	}
}
