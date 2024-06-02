package sdk

import (
	"fmt"
)

const (
	NetworkTestnet = "testnet"
	NetworkMainnet = "mainnet"

	ChainIDTestnet = "atlantic-2"
	ChainIDMainnet = "pacific-1"
)

type Config struct {
	Network  string
	GRPCHost string // "grpc.atlantic-2.seinetwork.io:443"
	RPCHost  string // "https://rpc.atlantic-2.seinetwork.io"

	SignerName     string
	SignerMnemonic string

	chainID string

	InsecureGRPC bool
	UseBasicAuth bool
}

func (cfg *Config) Validate() error {
	if cfg.Network != NetworkTestnet && cfg.Network != NetworkMainnet {
		return fmt.Errorf("invalid Network")
	}

	if cfg.Network == "" || cfg.RPCHost == "" || cfg.GRPCHost == "" {
		return fmt.Errorf("empty Network (%s) or RPCHost (%s) or GRPCHost (%s)", cfg.Network, cfg.RPCHost, cfg.GRPCHost)
	}

	if cfg.SignerName == "" || cfg.SignerMnemonic == "" {
		return fmt.Errorf("empty SignerName (%s) or SignerMnemonic", cfg.SignerName)
	}

	if cfg.Network == NetworkTestnet {
		cfg.chainID = ChainIDTestnet
	}

	if cfg.Network == NetworkMainnet {
		cfg.chainID = ChainIDMainnet
	}

	return nil
}
