package sei_sdk

import (
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	txf "github.com/cosmos/cosmos-sdk/client/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (c *Client) AddSigner(keyringUID, key string) {
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

	c.txFactory = txFactory

	sgn := &sign{
		ctx:     clientCtx,
		canSign: clientCtx.Keyring != nil,
		sender:  senderInfo.GetName(),
	}

	c.accNum, c.accSeq, err = txFactory.AccountRetriever().GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
	if err != nil {

	}

	c.txFactory = txFactory
	c.sign = sgn

	go func(cl *Client) {
		t := time.NewTicker(defaultTimeoutHeightSyncInterval)
		defer t.Stop()

		for {
			block, err := c.sign.ctx.Client.Block(c.cancelCtx, nil)
			if err != nil {
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
	}(c)
}
