package sdk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	txf "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codecTypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
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
	"github.com/sirupsen/logrus"
)

const (
	DefaultGasPriceWithDenom = "0.1usei"
	Bech32PrefixAccAddr      = "sei"
	Bech32PrefixAccPub       = "seipub"
)

type Client struct { //nolint:govet
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
	bankQueryClient banktypes.QueryClient

	// Accounts counter
	accNum uint64
	accSeq uint64

	// Logger
	logger *logrus.Entry
}

type sign struct {
	ctx    client.Context
	sender string
}

func NewClient(cfg Config, logger *logrus.Entry) (c *Client, err error) { //nolint:gocritic
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	config := sdktypes.GetConfig()
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

	cosmosKeyring := keyring.NewInMemory()
	path := hd.CreateHDPath(118, 0, 0).String()

	senderInfo, err := cosmosKeyring.NewAccount(cfg.SignerName, cfg.SignerMnemonic, "", path, hd.Secp256k1)
	if err != nil {
		return nil, fmt.Errorf("cosmosKeyring.NewAccount error: %w", err)
	}

	marshaller := codec.NewProtoCodec(interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaller, []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})

	tmClient, err := client.NewClientFromNode(cfg.RPCHost)
	if err != nil {
		return nil, fmt.Errorf("NewClientFromNode error: %w", err)
	}

	clientCtx := client.Context{
		ChainID:       cfg.chainID,
		BroadcastMode: flags.BroadcastAsync,
		TxConfig:      txConfig,
	}.WithKeyring(cosmosKeyring).WithFromAddress(senderInfo.GetAddress()).
		WithFromName(senderInfo.GetName()).WithFrom(senderInfo.GetName()).
		WithAccountRetriever(authtypes.AccountRetriever{}).WithClient(tmClient).
		WithInterfaceRegistry(interfaceRegistry)

	txFactory := new(txf.Factory).
		WithKeybase(clientCtx.Keyring).
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithSimulateAndExecute(true).
		WithGasAdjustment(1.1).
		WithChainID(clientCtx.ChainID).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithGasPrices(DefaultGasPriceWithDenom)

	sgn := &sign{
		ctx:    clientCtx,
		sender: senderInfo.GetAddress().String(),
	}

	accNum, accSeq, err := txFactory.AccountRetriever().GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
	if err != nil {
		return nil, fmt.Errorf("GetAccountNumberSequence: %w", err)
	}

	conn, err := getGRPCConn(cfg)
	if err != nil {
		return nil, fmt.Errorf("getGRPCConn: %s", err)
	}

	cancelCtx, cancelFn := context.WithCancel(context.Background())
	c = &Client{
		conn:      conn,
		syncMux:   new(sync.Mutex),
		cancelCtx: cancelCtx,
		cancelFn:  cancelFn,

		txFactory:       txFactory,
		txClient:        txtypes.NewServiceClient(conn),
		wasmQueryClient: wasmtypes.NewQueryClient(conn),
		bankQueryClient: banktypes.NewQueryClient(conn),

		logger: logger,
		accNum: accNum,
		accSeq: accSeq,
		sign:   sgn,
	}

	go func() {
		t := time.NewTicker(defaultTimeoutHeightSyncInterval)
		defer t.Stop()

		for {
			block, err := clientCtx.Client.Block(c.cancelCtx, nil)
			if err != nil {
				c.logger.Errorf("failed to get current block: %s", err)

				continue
			}

			c.txFactory.WithTimeoutHeight(uint64(block.Block.Height) + defaultTimeoutHeight)

			select {
			case <-c.cancelCtx.Done():
				return
			case <-t.C:
				continue
			}
		}
	}()

	return c, nil
}

func (c *Client) GetSignerAddress() string {
	return c.sign.sender
}
