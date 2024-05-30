package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

func (c *Client) asyncBroadcastMsg(msgs ...sdktypes.Msg) (*txtypes.BroadcastTxResponse, error) {
	ctx := context.Background()
	c.syncMux.Lock()
	defer c.syncMux.Unlock()

	sequence := c.getAccSeq()
	c.txFactory = c.txFactory.WithSequence(sequence)
	c.txFactory = c.txFactory.WithAccountNumber(c.accNum)

	c.logger.Debugf("asyncBroadcastMsg: send with seq %d", sequence)
	res, err := c.broadcastTx(ctx, c.txFactory, msgs...)
	if err != nil {
		for i := range 5 {
			if err == nil {
				break
			}

			if strings.Contains(err.Error(), "account sequence mismatch") {
				err = c.syncNonce()
				if err != nil {
					c.logger.Warnf("syncNonce failed: %s", err)
					continue
				}

				prevSeq := sequence
				sequence = c.getAccSeq()

				c.txFactory = c.txFactory.WithSequence(sequence)
				c.txFactory = c.txFactory.WithAccountNumber(c.accNum)

				c.logger.Warnf("broadcastTx retry: %d; prevSeq %d, curSeq %d; err %s", i, prevSeq, sequence, err)

				res, err = c.broadcastTx(ctx, c.txFactory, msgs...)
				continue
			}

			break
		}

		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (c *Client) broadcastTx(ctx context.Context, txf tx.Factory, msgs ...sdktypes.Msg) (resp *txtypes.BroadcastTxResponse, err error) { //nolint:gocritic
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

func (c *Client) getAccSeq() uint64 {
	defer func() {
		c.accSeq++
	}()
	return c.accSeq
}

func (c *Client) syncNonce() error {
	num, seq, err := c.txFactory.AccountRetriever().GetAccountNumberSequence(c.sign.ctx, c.sign.ctx.GetFromAddress())
	c.logger.Debugf("syncNonce: c.accNum %d, num %d, prevSeq %d, curSeq %d", c.accNum, num, c.accSeq, seq)

	if err != nil {
		return fmt.Errorf("GetAccountNumberSequence: %w", err)
	} else if num != c.accNum {
		return nil
	}

	c.accSeq = seq

	return nil
}
