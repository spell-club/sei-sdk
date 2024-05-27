package seisdk

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/sirupsen/logrus"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	txf "github.com/cosmos/cosmos-sdk/client/tx"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
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
	DefaultGasPriceWithDenom = "0.1usei"
	Bech32PrefixAccAddr      = "sei"
	Bech32PrefixAccPub       = "seipub"
)

type Client struct {
	// Sign for transactions
	sign *sign

	// Conn and sync services
	conn      *grpc.ClientConn
	syncMux   *sync.Mutex
	cancelCtx context.Context
	cancelFn  func()

	// Execution clients
	txFactory       txf.Factory
	txClient        txtypes.ServiceClient
	wasmQueryClient wasmtypes.QueryClient
	tmClient        *rpchttp.HTTP

	// Accounts counter
	accNum uint64
	accSeq uint64

	// Some config data
	rpcHost  string
	chainID  string
	contract string

	// Interfaces that we will reuse in AddSign
	interfaceRegistry codecTypes.InterfaceRegistry
	logger            *logrus.Entry
}

type sign struct {
	ctx    client.Context
	sender string
}

func NewClient(cfg Config, logger *logrus.Entry) (c *Client, err error) {
	if cfg.Network != "testnet" && cfg.Network != "mainnet" {
		return c, fmt.Errorf("invalid network: %s. Can be 'testnet' or 'mainnet'", cfg.Network)
	}

	config := sdk.GetConfig()
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
		return nil, fmt.Errorf("NewClientFromNode error: %w", err)
	}

	conn, err := grpc.NewClient(cfg.GRPCHost, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	if err != nil {
		return nil, fmt.Errorf("grpc.Dial: %s %s", cfg.GRPCHost, err)
	}

	cancelCtx, cancelFn := context.WithCancel(context.Background())

	c = &Client{
		conn:      conn,
		syncMux:   new(sync.Mutex),
		cancelCtx: cancelCtx,
		cancelFn:  cancelFn,

		txClient:        txtypes.NewServiceClient(conn),
		wasmQueryClient: wasmtypes.NewQueryClient(conn),
		tmClient:        tmClient,

		rpcHost:  cfg.RPCHost,
		chainID:  cfg.ChainID,
		contract: cfg.Contract,

		interfaceRegistry: interfaceRegistry,
		logger:            logger,
	}

	return c, nil
}
