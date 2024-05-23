package sei_sdk

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	std "github.com/cosmos/cosmos-sdk/std"
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
)

type Client struct {
	conn            *grpc.ClientConn
	syncMux         *sync.Mutex
	cancelCtx       context.Context
	cancelFn        func()
	txClient        txtypes.ServiceClient
	wasmQueryClient wasmtypes.QueryClient

	interfaceRegistry codecTypes.InterfaceRegistry

	nodeURI string
	chainID string
}

func NewClient(cfg ClientConfig) (c *Client, err error) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("sei", "seipub")
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

	conn, err := grpc.NewClient(cfg.RPCAddress, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	if err != nil {
		return nil, fmt.Errorf("grpc.Dial: %s %s", cfg.RPCAddress, err)
	}

	cancelCtx, cancelFn := context.WithCancel(context.Background())

	c = &Client{
		conn:      conn,
		syncMux:   new(sync.Mutex),
		cancelCtx: cancelCtx,
		cancelFn:  cancelFn,

		txClient:        txtypes.NewServiceClient(conn),
		wasmQueryClient: wasmtypes.NewQueryClient(conn),

		interfaceRegistry: interfaceRegistry,

		nodeURI: cfg.NodeURI,
	}

	return c, nil
}
