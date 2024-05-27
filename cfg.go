package seisdk

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

const (
	SingleTxMode = "single"
	BatchTxMode  = "batch"
)

type Config struct {
	Network  string
	Contract string
	TxMode   string
	ChainID  string
	GRPCHost string // "grpc.atlantic-2.seinetwork.io:443"
	RPCHost  string // "https://rpc.atlantic-2.seinetwork.io"
}

func (cfg *Config) Validate() error {
	if cfg.Network != "testnet" && cfg.Network != "mainnet" && cfg.Network != "devnet" {
		return fmt.Errorf("invalid Network")
	}

	hrp, _, err := bech32.DecodeAndConvert(cfg.Contract)
	if err != nil || hrp != Bech32PrefixAccAddr {
		return fmt.Errorf("invalid Contract")
	}

	if cfg.TxMode != SingleTxMode && cfg.TxMode != BatchTxMode {
		return fmt.Errorf("invalid TxMode: %s; Possible values: %s, %s", cfg.TxMode, SingleTxMode, BatchTxMode)
	}

	if cfg.Network == "" || cfg.RPCHost == "" || cfg.Contract == "" {
		return fmt.Errorf("invalid Network (%s) or RPCHost (%s) or GRPCHost (%s)", cfg.Network, cfg.RPCHost, cfg.GRPCHost)
	}

	return nil
}
