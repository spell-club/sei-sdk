package sei_sdk

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

const (
	defaultTimeoutHeight             = 20
	defaultTimeoutHeightSyncInterval = 10 * time.Second
	msgBatchLen                      = 50
)

func (c *Client) SendTx(msgs []string) (string, error) {
	if len(msgs) == 0 {
		return "", errors.New("message is empty")
	}
	if len(msgs) > msgBatchLen {
		return "", errors.New("too many messages")
	}

	txResult, err := c.asyncBroadcastMsg(Map(msgs, func(d string) cosmostypes.Msg {
		return &wasmtypes.MsgExecuteContract{
			Sender:   c.sign.sender,
			Contract: c.contract,
			Msg:      []byte(d),
		}
	})...)
	if err != nil {
		if strings.Contains(err.Error(), "is greater than max gas") && len(msgs) > 2 {
			var txHashR string

			for _, chunk := range Chunk(msgs, len(msgs)/2+1) {
				txHashR, err = c.SendTx(chunk)
				if err != nil {
					return "", fmt.Errorf("SendTx recursive call: %s", err)
				}
			}

			return txHashR, nil
		}

		return "", fmt.Errorf("AsyncBroadcastMsg: %s", err)
	}

	if txResult == nil || txResult.GetTxResponse() == nil {
		return "", fmt.Errorf("txResult is nil: %v", txResult)
	}

	return txResult.GetTxResponse().TxHash, nil
}

func (c *Client) asyncBroadcastMsg(msgs ...sdk.Msg) (*txtypes.BroadcastTxResponse, error) {
	log.Printf("starting async broadcast")

	ctx := context.Background()
	c.syncMux.Lock()
	defer c.syncMux.Unlock()

	sequence := c.getAccSeq()
	c.txFactory = c.txFactory.WithSequence(sequence)
	c.txFactory = c.txFactory.WithAccountNumber(c.accNum)

	res, err := c.broadcastTx(ctx, c.txFactory, msgs...)
	if err != nil {
		if strings.Contains(err.Error(), "account sequence mismatch") {
			c.syncNonce()

			sequence = c.getAccSeq()
			c.txFactory = c.txFactory.WithSequence(sequence)
			c.txFactory = c.txFactory.WithAccountNumber(c.accNum)

			res, err = c.broadcastTx(ctx, c.txFactory, msgs...)
		}
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (c *Client) broadcastTx(ctx context.Context, txf tx.Factory, msgs ...sdk.Msg) (resp *txtypes.BroadcastTxResponse, err error) {
	txf, err = c.prepareFactory(c.sign.ctx, txf)
	if err != nil {
		return nil, fmt.Errorf("c.prepareFactory: %s", err)
	}

	simTxBytes, err := txf.BuildSimTx(msgs...)
	if err != nil {
		return nil, fmt.Errorf("txf.BuildSimTx: %s", err)
	}

	simRes, err := c.txClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: simTxBytes})
	if err != nil {
		return nil, fmt.Errorf("c.txClient.Simulate: %s", err)
	}

	adjustedGas := uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GetGasUsed()))
	txf = txf.WithGas(adjustedGas)

	txn, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, fmt.Errorf("BuildUnsignedTx: %s", err)
	}

	txn.SetFeeGranter(c.sign.ctx.GetFeeGranterAddress())
	err = tx.Sign(txf, c.sign.ctx.GetFromName(), txn, true)
	if err != nil {
		return nil, fmt.Errorf("tx.Sign: %s", err)
	}

	txBytes, err := c.sign.ctx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return nil, fmt.Errorf("c.ctx.TxConfig.TxEncoder: %s", err)
	}

	req := txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	}

	resp, err = c.txClient.BroadcastTx(ctx, &req)
	if err != nil {
		return resp, fmt.Errorf("BroadcastTx: %s", err)
	}

	if resp.GetTxResponse() == nil {
		return resp, fmt.Errorf("resp.GetTxResponse(): %+v", resp)
	}

	if resp.GetTxResponse().RawLog != "" {
		return resp, fmt.Errorf("resp.GetTxResponse().RawLog: %s", resp.GetTxResponse().RawLog)
	}

	return resp, nil
}

func (*Client) prepareFactory(clientCtx client.Context, txf tx.Factory) (tx.Factory, error) {
	from := clientCtx.GetFromAddress()

	if err := txf.AccountRetriever().EnsureExists(clientCtx, from); err != nil {
		return txf, err
	}

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err := txf.AccountRetriever().GetAccountNumberSequence(clientCtx, from)
		if err != nil {
			return txf, err
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}

	return txf, nil
}

func (c *Client) getAccSeq() uint64 {
	defer func() {
		c.accSeq++
	}()
	return c.accSeq
}

func (c *Client) syncNonce() {
	num, seq, err := c.txFactory.AccountRetriever().GetAccountNumberSequence(c.sign.ctx, c.sign.ctx.GetFromAddress())
	if err != nil {
		return
	} else if num != c.accNum {
		return
	}

	c.accSeq = seq
}
