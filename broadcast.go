package sdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// broadcastTx signs and broadcasts tx to the network
// it also does several other things
// - retrieves the proper acc sequence via GetAccountNumberSequence
// - runs the simulation via Simulate
// - adjusts Gas
func (c *Client) broadcastTx(ctx context.Context, sgn signer, msgs ...sdktypes.Msg) (resp *txtypes.BroadcastTxResponse, err error) {
	if !c.canSign {
		return resp, errors.New("can't sign. Add signature before sending tx")
	}
	if sgn.address.Empty() {
		return resp, errors.New("empty signer")
	}

	num, seq, err := c.clientCtx.AccountRetriever.GetAccountNumberSequence(c.clientCtx, sgn.address.Bytes())
	if err != nil {
		return resp, fmt.Errorf("GetAccountNumberSequence: %s", err)
	}
	txf := c.txFactory.WithSequence(seq).WithAccountNumber(num)

	simTxBytes, err := txf.BuildSimTx(msgs...)
	if err != nil {
		return resp, fmt.Errorf("BuildSimTx: %s", err)
	}
	simRes, err := c.txClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: simTxBytes})
	if err != nil {
		return resp, fmt.Errorf("Simulate: %s", err)
	}

	adjustedGas := uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GetGasUsed()))
	txf = txf.WithGas(adjustedGas)
	txn, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return resp, fmt.Errorf("BuildUnsignedTx: %s", err)
	}

	err = tx.Sign(txf, sgn.name, txn, true)
	if err != nil {
		return resp, fmt.Errorf("Sign: %s", err)
	}

	txBytes, err := c.clientCtx.TxConfig.TxEncoder()(txn.GetTx())
	if err != nil {
		return resp, fmt.Errorf("TxEncoder: %s", err)
	}

	resp, err = c.txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	})
	if err != nil {
		return resp, fmt.Errorf("BroadcastTx: %s", err)
	}
	if resp.GetTxResponse() == nil {
		return resp, errors.New("GetTxResponse == nil")
	}
	if rawLog := resp.GetTxResponse().RawLog; rawLog != "" {
		return resp, fmt.Errorf("txHash.GetTxResponse().RawLog: %s", rawLog)
	}
	if resp.GetTxResponse().TxHash == "" {
		return resp, errors.New("empty TxHash")
	}

	return
}
