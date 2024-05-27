package seisdk

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"

	txf "github.com/cosmos/cosmos-sdk/client/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// AddSigner TODO: one template for all signers
func (c *Client) AddSigner(name, mnemonic string) error {
	tmClient, err := client.NewClientFromNode(c.rpcHost)
	if err != nil {
		return fmt.Errorf("NewClientFromNode error: %w", err)
	}

	cosmosKeyring := keyring.NewInMemory()
	path := hd.CreateHDPath(118, 0, 0).String()

	senderInfo, err := cosmosKeyring.NewAccount(name, mnemonic, "", path, hd.Secp256k1)
	if err != nil {
		return fmt.Errorf("cosmosKeyring.NewAccount error: %w", err)
	}

	marshaller := codec.NewProtoCodec(c.interfaceRegistry)
	txConfig := tx.NewTxConfig(marshaller, []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT})

	clientCtx := client.Context{
		ChainID:       c.chainID,
		BroadcastMode: flags.BroadcastAsync,
		TxConfig:      txConfig,
	}.WithKeyring(cosmosKeyring).WithFromAddress(senderInfo.GetAddress()).
		WithFromName(senderInfo.GetName()).WithFrom(senderInfo.GetName()).
		WithAccountRetriever(authtypes.AccountRetriever{}).WithClient(tmClient).
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
		ctx:    clientCtx,
		sender: senderInfo.GetAddress().String(),
	}

	c.accNum, c.accSeq, err = txFactory.AccountRetriever().GetAccountNumberSequence(clientCtx, clientCtx.GetFromAddress())
	if err != nil {
		return fmt.Errorf("GetAccountNumberSequence error: %w", err)
	}

	c.sign = sgn

	go func(cl *Client) {
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
	}(c)

	return nil
}
