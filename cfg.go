package sdk

import (
	"fmt"
)

const (
	SingleTxMode = "single"
	BatchTxMode  = "batch"

	NetworkTestnet = "testnet"
	NetworkMainnet = "mainnet"

	ChainIDTestnet = "atlantic-2"
	ChainIDMainnet = "pacific-1"
)

type Config struct {
	Network  string
	TxMode   string
	GRPCHost string // "grpc.atlantic-2.seinetwork.io:443"
	RPCHost  string // "https://rpc.atlantic-2.seinetwork.io"

	SignerName     string
	SignerMnemonic string

	BuildAuthBasedOnRPCHost bool

	chainID string
}

func (cfg *Config) Validate() error {
	if cfg.Network != NetworkTestnet && cfg.Network != NetworkMainnet {
		return fmt.Errorf("invalid Network")
	}

	if cfg.TxMode != SingleTxMode && cfg.TxMode != BatchTxMode {
		return fmt.Errorf("invalid TxMode: %s; Possible values: %s, %s", cfg.TxMode, SingleTxMode, BatchTxMode)
	}

	if cfg.Network == "" || cfg.RPCHost == "" {
		return fmt.Errorf("empty Network (%s) or RPCHost (%s)", cfg.Network, cfg.RPCHost)
	}

	if cfg.GRPCHost == "" && !cfg.BuildAuthBasedOnRPCHost {
		return fmt.Errorf("empty GRPCHost (%s)", cfg.GRPCHost)
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
