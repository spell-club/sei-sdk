package sdk

import (
	"errors"
	"fmt"
	"sync"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	txf "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	feegranttypes "github.com/cosmos/cosmos-sdk/x/feegrant"
	paramproposaltypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const (
	// DefaultGasPriceWithDenom defines the default gas price denomination
	DefaultGasPriceWithDenom = "0.1usei"
	// Bech32PrefixAccAddr defines the Bech32 prefix for account addresses
	Bech32PrefixAccAddr = "sei"
	// Bech32PrefixAccPub defines the Bech32 prefix for account public keys
	Bech32PrefixAccPub = "seipub"
)

// Client represents a Cosmos SDK client for interacting with a blockchain node
type Client struct {
	// Execution clients for sending transactions and querying data
	// Execution clients
	txFactory       txf.Factory
	txClient        txtypes.ServiceClient
	wasmQueryClient wasmtypes.QueryClient
	bankQueryClient banktypes.QueryClient
	clientCtx       client.Context

	signers map[string]*signer

	canSign bool
}

// signer holds information about a signer
type signer struct {
	syncMux *sync.Mutex
	address cosmosTypes.Address
	name    string
}

// NewClient creates a new Cosmos SDK client
func NewClient(cfg Config) (c *Client, err error) {
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	config := cosmosTypes.GetConfig()
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.Seal()

	interfaceRegistry := codecTypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authtypes.RegisterInterfaces(interfaceRegistry)
	authztypes.RegisterInterfaces(interfaceRegistry)
	vestingtypes.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	crisistypes.RegisterInterfaces(interfaceRegistry)
	distributiontypes.RegisterInterfaces(interfaceRegistry)
	evidencetypes.RegisterInterfaces(interfaceRegistry)
	paramproposaltypes.RegisterInterfaces(interfaceRegistry)
	slashingtypes.RegisterInterfaces(interfaceRegistry)
	stakingtypes.RegisterInterfaces(interfaceRegistry)
	upgradetypes.RegisterInterfaces(interfaceRegistry)
	feegranttypes.RegisterInterfaces(interfaceRegistry)

	tmClient, err := client.NewClientFromNode(cfg.RPCHost)
	if err != nil {
		return nil, fmt.Errorf("NewClientFromNode: %s", err)
	}

	clientCtx := client.Context{}.
		WithTxConfig(tx.NewTxConfig(codec.NewProtoCodec(interfaceRegistry), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})).
		WithChainID(string(cfg.ChainID)).
		WithKeyring(keyring.NewInMemory()).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithClient(tmClient).
		WithInterfaceRegistry(interfaceRegistry)

	txFactory := txf.Factory{}.
		WithKeybase(clientCtx.Keyring).
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithSimulateAndExecute(true).
		WithGasAdjustment(1.1).
		WithChainID(clientCtx.ChainID).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithGasPrices(DefaultGasPriceWithDenom)

	conn, err := getGRPCConn(cfg)
	if err != nil {
		return nil, fmt.Errorf("getGRPCConn: %s", err)
	}

	return &Client{
		txFactory:       txFactory,
		txClient:        txtypes.NewServiceClient(conn),
		wasmQueryClient: wasmtypes.NewQueryClient(conn),
		bankQueryClient: banktypes.NewQueryClient(conn),

		clientCtx: clientCtx,
		signers:   make(map[string]*signer),
	}, nil
}

// GetSignerAddresses returns a list of addresses for every added signer
func (c *Client) GetSignerAddresses() (res []string) {
	for _, s := range c.signers {
		res = append(res, s.address.String())
	}

	return
}

// getSigner returns signer by name
func (c *Client) getSigner(name string) (*signer, error) {
	sgn, ok := c.signers[name]
	if !ok {
		return nil, fmt.Errorf("signer with name %s not added", name)
	}

	return sgn, nil
}

// AddSigner adds signer by name, so it can be later used for signing
func (c *Client) AddSigner(name, mnemonic string) error {
	if name == "" {
		return errors.New("empty name")
	}
	if mnemonic == "" {
		return errors.New("empty mnemonic")
	}

	if _, ok := c.signers[name]; ok {
		return fmt.Errorf("duplicate signer %s", name)
	}

	path := hd.CreateHDPath(118, 0, 0).String()
	signerInfo, err := c.clientCtx.Keyring.NewAccount(name, mnemonic, "", path, hd.Secp256k1)
	if err != nil {
		return fmt.Errorf("NewAccount: %w", err)
	}

	c.signers[name] = &signer{
		syncMux: &sync.Mutex{},
		address: signerInfo.GetAddress(),
		name:    name,
	}
	c.canSign = true

	return nil
}
