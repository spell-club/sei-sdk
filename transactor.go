package sei_sdk

import (
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	txf "github.com/cosmos/cosmos-sdk/client/tx"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type Transactor struct {
	ctx       client.Context
	txFactory txf.Factory
	canSign   bool
	accNum    uint64
	accSeq    uint64

	*Client
}

func (c *Client) NewTransactor(keyringUID, key string) *Transactor {
	tmClient, err := client.NewClientFromNode(c.nodeURI)
	if err != nil {

	}

	cosmosKeyring := keyring.NewInMemory()
	path := hd.CreateHDPath(118, 0, 0).String()

	senderInfo, err := cosmosKeyring.NewAccount(keyringUID, key, "", path, hd.Secp256k1)
	if err != nil {

	}

	std.RegisterInterfaces(c.interfaceRegistry)
	marshaller := codec.NewProtoCodec(c.interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaller, []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})

	clientCtx := client.Context{
		ChainID:       c.chainID,
		BroadcastMode: flags.BroadcastAsync,
		TxConfig:      txConfig,
	}.WithKeyring(cosmosKeyring).WithFromAddress(senderInfo.GetAddress()).
		WithFromName(senderInfo.GetName()).WithFrom(senderInfo.GetName()).
		WithNodeURI(c.nodeURI).WithAccountRetriever(authtypes.AccountRetriever{}).WithClient(tmClient).
		WithInterfaceRegistry(c.interfaceRegistry)

	txFactory := new(txf.Factory).
		WithKeybase(clientCtx.Keyring).
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithSimulateAndExecute(true).
		WithGasAdjustment(1.1).
		WithChainID(clientCtx.ChainID).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithGasPrices(DefaultGasPriceWithDenom)

	transactor := &Transactor{
		ctx:       clientCtx,
		txFactory: txFactory,
		canSign:   clientCtx.Keyring != nil,

		Client: c,
	}

	transactor.accNum, transactor.accSeq, err = txFactory.AccountRetriever().GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
	if err != nil {

	}

	go func(tx *Transactor) {
		t := time.NewTicker(defaultTimeoutHeightSyncInterval)
		defer t.Stop()

		for {
			block, err := tx.ctx.Client.Block(c.cancelCtx, nil)
			if err != nil {
				continue
			}

			tx.txFactory.WithTimeoutHeight(uint64(block.Block.Height) + defaultTimeoutHeight)

			select {
			case <-c.cancelCtx.Done():
				return
			case <-t.C:
				continue
			}
		}
	}(transactor)

	return transactor
}

func (c *Transactor) SendTx(sender, contract string, msgs []string) (string, error) {
	if len(msgs) == 0 {
		return "", errors.New("message is empty")
	}
	if len(msgs) > 100 {
		return "", errors.New("too many messages")
	}

	result, err := c.asyncBroadcastMsg(Map(msgs, func(d string) cosmostypes.Msg {
		return &wasmtypes.MsgExecuteContract{
			Sender:   sender,
			Contract: contract,
			Msg:      []byte(d),
		}
	})...)
	if err != nil {
		return "", fmt.Errorf("AsyncBroadcastMsg: %s", err)
	}
	if result == nil || result.GetTxResponse() == nil {
		return "", fmt.Errorf("result is nil: %v", result)
	}

	return result.GetTxResponse().TxHash, nil
}
