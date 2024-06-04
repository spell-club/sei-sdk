package sdk

import (
	"errors"
	"fmt"
)

const (
	ChainIDTestnet ChainIDType = "atlantic-2"
	ChainIDMainnet ChainIDType = "pacific-1"
)

type (
	ChainIDType string
	Config      struct {
		GRPCHost string
		RPCHost  string

		ChainID ChainIDType

		InsecureGRPC bool
		UseBasicAuth bool
	}
)

func (cfg *Config) Validate() error {
	if cfg.ChainID != ChainIDTestnet && cfg.ChainID != ChainIDMainnet {
		return fmt.Errorf("invalid ChainID: %s. Possible values: %s, %s", cfg.ChainID, ChainIDMainnet, ChainIDTestnet)
	}
	if cfg.RPCHost == "" {
		return errors.New("empty RPCHost")
	}
	if cfg.GRPCHost == "" {
		return errors.New("empty GRPCHost")
	}

	return nil
}
