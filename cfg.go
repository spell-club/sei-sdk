package sdk

import (
	"fmt"
)

const (
	SingleTxMode = "single"
	BatchTxMode  = "batch"
)

type Config struct {
	Network  string
	TxMode   string
	ChainID  string
	GRPCHost string // "grpc.atlantic-2.seinetwork.io:443"
	RPCHost  string // "https://rpc.atlantic-2.seinetwork.io"

	SignerName     string
	SignerMnemonic string
}

func (cfg *Config) Validate() error {
	if cfg.Network != "testnet" && cfg.Network != "mainnet" && cfg.Network != "devnet" {
		return fmt.Errorf("invalid Network")
	}

	if cfg.TxMode != SingleTxMode && cfg.TxMode != BatchTxMode {
		return fmt.Errorf("invalid TxMode: %s; Possible values: %s, %s", cfg.TxMode, SingleTxMode, BatchTxMode)
	}

	if cfg.Network == "" || cfg.RPCHost == "" || cfg.GRPCHost == "" {
		return fmt.Errorf("empty Network (%s) or RPCHost (%s) or GRPCHost (%s)", cfg.Network, cfg.RPCHost, cfg.GRPCHost)
	}

	if cfg.SignerName == "" || cfg.SignerMnemonic == "" {
		return fmt.Errorf("empty SignerName (%s) or SignerMnemonic", cfg.SignerName)
	}

	return nil
}
