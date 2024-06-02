package sdk

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/tendermint/tendermint/rpc/coretypes"
)

const (
	getTxAttempts = 20
	failedMsgKey  = "failed_msgs"
)

func (c *Client) asyncBroadcastMsg(ctx context.Context, msgs ...sdktypes.Msg) (res string, err error) {
	c.syncMux.Lock()
	defer c.syncMux.Unlock()

	sequence := c.getAccSeq()
	c.txFactory = c.txFactory.WithSequence(sequence)

	res, err = c.broadcastTx(ctx, c.txFactory, msgs...)
	if err != nil {
		for i := range 5 {
			if err == nil {
				break
			}

			if strings.Contains(err.Error(), "account sequence mismatch") {
				if err := c.syncNonce(); err != nil {
					if c.logger != nil {
						c.logger.Warnf("broadcastTx: syncNonce failed: %s", err)
					}
					continue
				}

				sequence = c.getAccSeq()
				c.txFactory = c.txFactory.WithSequence(sequence)

				if c.logger != nil {
					c.logger.Infof("broadcastTx retry: %d; curSeq %d; err %s", i, sequence, err)
				}

				res, err = c.broadcastTx(ctx, c.txFactory, msgs...)
				continue
			}

			break
		}
	}

	return res, err
}

func (c *Client) broadcastTx(ctx context.Context, txf tx.Factory, msgs ...sdktypes.Msg) (txHash string, err error) { //nolint:gocritic
	txf, err = c.prepareFactory(c.sign.ctx, txf)
	if err != nil {
		return txHash, fmt.Errorf("c.prepareFactory: %s", err)
	}

	simTxBytes, err := txf.BuildSimTx(msgs...)
	if err != nil {
		return txHash, fmt.Errorf("txf.BuildSimTx: %s", err)
	}
	simRes, err := c.txClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: simTxBytes})
	if err != nil {
		return txHash, fmt.Errorf("c.txClient.Simulate: %s", err)
	}

	adjustedGas := uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GetGasUsed()))
	txf = txf.WithGas(adjustedGas)
	txn, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return txHash, fmt.Errorf("BuildUnsignedTx: %s", err)
	}

	txn.SetFeeGranter(c.sign.ctx.GetFeeGranterAddress())
	err = tx.Sign(txf, c.sign.ctx.GetFromName(), txn, true)
	if err != nil {
		return txHash, fmt.Errorf("tx.Sign: %s", err)
	}

	txBytes, err := c.sign.ctx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return txHash, fmt.Errorf("c.ctx.TxConfig.TxEncoder: %s", err)
	}

	resp, err := c.txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	})
	if err != nil {
		return txHash, fmt.Errorf("BroadcastTx: %s", err)
	}
	if resp.GetTxResponse() == nil {
		return txHash, errors.New("resp.GetTxResponse == nil")
	}
	if rawLog := resp.GetTxResponse().RawLog; rawLog != "" {
		return txHash, fmt.Errorf("txHash.GetTxResponse().RawLog: %s", rawLog)
	}

	return resp.GetTxResponse().TxHash, nil
}

func (*Client) prepareFactory(clientCtx client.Context, txf tx.Factory) (tx.Factory, error) { //nolint:gocritic
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

func (c *Client) getAccSeq() (res uint64) {
	res = c.accSeq
	c.accSeq++
	return
}

func (c *Client) syncNonce() error {
	num, seq, err := c.txFactory.AccountRetriever().GetAccountNumberSequence(c.sign.ctx, c.sign.ctx.GetFromAddress())
	if err != nil {
		return fmt.Errorf("GetAccountNumberSequence: %w", err)
	} else if num != c.accNum {
		return fmt.Errorf("mismatch acc num %d %d", num, c.accNum)
	}

	c.accSeq = seq

	return nil
}

func (c *Client) GetTxByHashWithRetry(ctx context.Context, txHash string) (failedIndexes string, err error) {
	cl, err := c.sign.ctx.GetNode()
	if err != nil {
		return failedIndexes, fmt.Errorf("GetNode: %s", err)
	}
	decodedTxHash, err := hex.DecodeString(txHash)
	if err != nil {
		return failedIndexes, fmt.Errorf("DecodeString: %s", err)
	}

	var txResp *coretypes.ResultTx
	for range getTxAttempts {
		select {
		case <-ctx.Done():
			return failedIndexes, context.Canceled
		default:
		}

		time.Sleep(time.Second)

		txResp, err = cl.Tx(ctx, decodedTxHash, true)
		if err != nil {
			if strings.Contains(err.Error(), "RPC error -32603") {
				continue
			}

			return failedIndexes, fmt.Errorf("txClient.GetTx: %w", err)
		}

		if txResp == nil {
			continue
		}

		if txResp.TxResult.Code != 0 {
			return failedIndexes, fmt.Errorf("non-zero code: %d", txResp.TxResult.Code)
		}

		break
	}

	if txResp == nil || len(txResp.Hash) == 0 {
		return failedIndexes, errors.New("fail to get tx after retries")
	}

	for _, e := range txResp.TxResult.Events {
		if e.Type != "wasm" {
			continue
		}
		for _, a := range e.Attributes {
			if string(a.Key) != failedMsgKey {
				continue
			}

			failedIndexes = string(a.Value)
		}
	}

	return failedIndexes, nil
}
