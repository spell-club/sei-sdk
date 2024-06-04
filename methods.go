package sdk

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tendermint/tendermint/rpc/coretypes"
)

const DefaultDenom = "usei"

func (c *Client) GetBankBalance(ctx context.Context, address, denom string) (*banktypes.QueryBalanceResponse, error) {
	req := &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	}
	return c.bankQueryClient.Balance(ctx, req)
}

func (c *Client) Execute(ctx context.Context, signerName, contractAddress, msg string) (resp *txtypes.BroadcastTxResponse, err error) {
	if msg == "" {
		return resp, errors.New("message is empty")
	}

	sgn, err := c.getSigner(signerName)
	if err != nil {
		return resp, err
	}

	resp, err = c.broadcastTx(ctx, sgn, &wasmtypes.MsgExecuteContract{
		Sender:   sgn.address.String(),
		Contract: contractAddress,
		Msg:      []byte(msg),
	})
	if err != nil {
		return resp, fmt.Errorf("broadcastTx: %s", err)
	}

	return
}

func (c *Client) Instantiate(ctx context.Context, signerName string, codeID uint64, label, instantiateMsg string, funds []sdktypes.Coin) (resp *txtypes.BroadcastTxResponse, err error) {
	if instantiateMsg == "" {
		return resp, errors.New("message code is empty")
	}
	if label == "" {
		return resp, errors.New("label is empty")
	}

	sgn, err := c.getSigner(signerName)
	if err != nil {
		return resp, err
	}

	message := &wasmtypes.MsgInstantiateContract{
		Sender: sgn.address.String(),
		Admin:  sgn.address.String(),
		Label:  label,
		CodeID: codeID,
		Msg:    []byte(instantiateMsg),
		Funds:  funds,
	}
	resp, err = c.broadcastTx(ctx, sgn, message)
	if err != nil {
		return resp, fmt.Errorf("broadcastTx: %s", err)
	}

	return
}

func (c *Client) GetTxByHash(ctx context.Context, txHash string, retries uint, sleepInterval time.Duration) (txResp *coretypes.ResultTx, err error) {
	cl, err := c.clientCtx.GetNode()
	if err != nil {
		return txResp, fmt.Errorf("GetNode: %s", err)
	}
	decodedTxHash, err := hex.DecodeString(txHash)
	if err != nil {
		return txResp, fmt.Errorf("DecodeString: %s", err)
	}

	if retries == 0 {
		retries = 1
	}
	for range retries {
		select {
		case <-ctx.Done():
			return txResp, context.Canceled
		default:
		}

		if sleepInterval != 0 {
			time.Sleep(sleepInterval)
		}

		txResp, err = cl.Tx(ctx, decodedTxHash, true)
		if err != nil {
			// tx not found
			if strings.Contains(err.Error(), "RPC error -32603") {
				continue
			}

			return txResp, fmt.Errorf("GetTx: %w", err)
		}

		if txResp == nil {
			continue
		}

		if txResp.TxResult.Code != 0 {
			return txResp, fmt.Errorf("non-zero code: %d", txResp.TxResult.Code)
		}

		break
	}

	if err != nil {
		return txResp, fmt.Errorf("GetTx: %w", err)
	}

	if txResp == nil || len(txResp.Hash) == 0 {
		return txResp, errors.New("fail to get tx after retries")
	}

	return
}
