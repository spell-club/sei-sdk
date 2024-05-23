package sei_sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
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
	defaultTimeoutHeightSyncInterval = 10 * time.Second
	defaultTimeoutHeight             = 20
	DefaultGasPriceWithDenom         = "0.1usei"
)

type Client struct {
	ctx       client.Context
	conn      *grpc.ClientConn
	txFactory tx.Factory

	syncMux *sync.Mutex

	cancelCtx context.Context
	cancelFn  func()

	accNum uint64
	accSeq uint64

	txClient        txtypes.ServiceClient
	wasmQueryClient wasmtypes.QueryClient

	canSign bool
}

func NewClient(cfg ClientConfig) (c *Client, sender string, err error) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("sei", "seipub")
	config.Seal()

	tmClient, err := client.NewClientFromNode(cfg.NodeURI)
	if err != nil {
		return nil, "", fmt.Errorf("http.New: %s", err)
	}

	cosmosKeyring := keyring.NewInMemory()
	path := hd.CreateHDPath(118, 0, 0).String()

	senderInfo, err := cosmosKeyring.NewAccount(cfg.KeyringUID, cfg.Key, "", path, hd.Secp256k1)
	if err != nil {
		return nil, "", fmt.Errorf("keyring.NewAccount: %s", err)
	}

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

	clientCtx := client.Context{
		ChainID:       cfg.ChainID,
		BroadcastMode: flags.BroadcastAsync,
		TxConfig: newTxConfig([]signing.SignMode{
			signing.SignMode_SIGN_MODE_DIRECT,
		}),
	}.WithKeyring(cosmosKeyring).WithFromAddress(senderInfo.GetAddress()).
		WithFromName(senderInfo.GetName()).WithFrom(senderInfo.GetName()).
		WithNodeURI(cfg.NodeURI).WithAccountRetriever(authtypes.AccountRetriever{}).WithClient(tmClient).
		WithInterfaceRegistry(interfaceRegistry)

	txFactory := newTxFactory(clientCtx)
	txFactory = txFactory.WithGasPrices(DefaultGasPriceWithDenom)

	conn, err := grpc.NewClient(cfg.RPCAddress, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	if err != nil {
		return nil, "", fmt.Errorf("grpc.Dial: %s %s", cfg.RPCAddress, err)
	}

	cancelCtx, cancelFn := context.WithCancel(context.Background())

	c = &Client{
		ctx:       clientCtx,
		conn:      conn,
		txFactory: txFactory,
		canSign:   clientCtx.Keyring != nil,
		syncMux:   new(sync.Mutex),
		cancelCtx: cancelCtx,
		cancelFn:  cancelFn,

		txClient:        txtypes.NewServiceClient(conn),
		wasmQueryClient: wasmtypes.NewQueryClient(conn),
	}

	c.accNum, c.accSeq, err = c.txFactory.AccountRetriever().GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
	if err != nil {
		return nil, "", fmt.Errorf("failed to get initial account num and seq: %s", err)
	}

	go c.syncTimeoutHeight()

	return
}
