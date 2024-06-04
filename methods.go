package sdk

import (
	"context"
	"encoding/hex"
	"encoding/json"
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

const (
	// DefaultDenom is the default denomination for Sei blockchain
	DefaultDenom = "usei"
)

// GetBankBalance queries a Cosmos SDK bank for the balance of a specific account denominated in a specific denom
func (c *Client) GetBankBalance(ctx context.Context, address, denom string) (*banktypes.QueryBalanceResponse, error) {
	// Create a QueryBalanceRequest struct with the provided address and denom
	req := &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	}
	// Call the bankQueryClient to query the balance
	return c.bankQueryClient.Balance(ctx, req)
}

// ExecuteJson simplifies sending an arbitrary JSON message to a Wasm contract
func (c *Client) ExecuteJson(ctx context.Context, signerName, contractAddress string, msg interface{}) (resp *txtypes.BroadcastTxResponse, err error) {
	// Marshal the provided message into a byte array
	marshalledMsg, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	// Delegate the execution to the Execute function with the marshalled message
	return c.Execute(ctx, signerName, contractAddress, string(marshalledMsg))
}

// Execute broadcasts a transaction to execute a message on a Wasm contract
func (c *Client) Execute(ctx context.Context, signerName, contractAddress, msg string) (resp *txtypes.BroadcastTxResponse, err error) {
	// Validate that the message is not empty
	if msg == "" {
		return resp, errors.New("message is empty")
	}
	// Retrieve the signer information for the provided signer name
	sgn, err := c.getSigner(signerName)
	if err != nil {
		return resp, err
	}
	// Create a MsgExecuteContract message with the signer address, contract address, and message
	message := &wasmtypes.MsgExecuteContract{
		Sender:   sgn.address.String(),
		Contract: contractAddress,
		Msg:      []byte(msg),
	}
	// Broadcast the transaction using the broadcastTx function
	resp, err = c.broadcastTx(ctx, sgn, message)
	if err != nil {
		return resp, fmt.Errorf("broadcastTx: %s", err)
	}

	return
}

// InstantiateJson simplifies sending an arbitrary JSON message as the instantiate message for a Wasm contract
func (c *Client) InstantiateJson(ctx context.Context, signerName string, codeID uint64, label string, instantiateMsg interface{}, funds []sdktypes.Coin) (resp *txtypes.BroadcastTxResponse, err error) {
	// Marshal the provided instantiate message into a byte array
	marshalledMsg, err := json.Marshal(instantiateMsg)
	if err != nil {
		return nil, err
	}
	// Delegate the instantiation to the Instantiate function with the marshalled message
	return c.Instantiate(ctx, signerName, codeID, label, string(marshalledMsg), funds)
}

// Instantiate broadcasts a transaction to instantiate a Wasm contract
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

// GetTxByHash retrieves transaction from the network. retries and sleepInterval params can be used to re-retrieve tx in case of error
func (c *Client) GetTxByHash(ctx context.Context, txHash string, retries uint, sleepInterval time.Duration) (txResp *coretypes.ResultTx, err error) {
	cl, err := c.clientCtx.GetNode()
	if err != nil {
		return txResp, fmt.Errorf("GetNode: %s", err)
	}

	decodedTxHash, err := hex.DecodeString(txHash)
	if err != nil {
		return txResp, fmt.Errorf("DecodeString: %s", err)
	}

	// first is request, after n retries
	for i := range retries + 1 {
		select {
		case <-ctx.Done():
			return txResp, context.Canceled
		default:
		}

		// do not sleep on first request
		if sleepInterval != 0 && i > 0 {
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
