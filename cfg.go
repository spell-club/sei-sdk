package sdk

import (
	"errors"
)

const (
	ChainIDTestnet chainID = "atlantic-2"
	ChainIDMainnet chainID = "pacific-1"
)

type (
	chainID string
	Config  struct {
		GRPCHost string
		RPCHost  string

		ChainID chainID

		InsecureGRPC bool
		UseBasicAuth bool
	}
)

func (cfg *Config) Validate() error {
	if cfg.RPCHost == "" {
		return errors.New("empty RPCHost")
	}
	if cfg.GRPCHost == "" {
		return errors.New("empty GRPCHost")
	}

	return nil
}
