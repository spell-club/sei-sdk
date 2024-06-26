package sdk

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	abci "github.com/tendermint/tendermint/abci/types"
	"strconv"
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

	searchByHeightQuery = `tx.height>%d AND tx.height<=%d AND wasm._contract_address CONTAINS '%s'`
	rangeSize           = 100_000
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

// ExecuteJSON simplifies sending an arbitrary JSON message to a Wasm contract
func (c *Client) ExecuteJSON(ctx context.Context, signerName, contractAddress string, msg interface{}) (resp *txtypes.BroadcastTxResponse, err error) {
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

// InstantiateJSON simplifies sending an arbitrary JSON message as the instantiate message for a Wasm contract
func (c *Client) InstantiateJSON(ctx context.Context, signerName string, codeID uint64, label string, instantiateMsg interface{}, funds []sdktypes.Coin) (resp *txtypes.BroadcastTxResponse, err error) {
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

// GetLatestHeight retrieves latest height from the network.
func (c *Client) GetLatestHeight(ctx context.Context) (int64, error) {
	tendermintNode, err := c.clientCtx.GetNode()
	if err != nil {
		return 0, err
	}

	resp, err := tendermintNode.Status(ctx)
	if err != nil {
		return 0, fmt.Errorf("tendermintNode.Status: %w", err)
	}

	return resp.SyncInfo.LatestBlockHeight, nil
}

// GetTxMetaResponseByHash retrieves transaction metadata response from the network. retries and sleepInterval params can be used to re-retrieve tx in case of error
func (c *Client) GetTxMetaResponseByHash(ctx context.Context, txHash string, retries uint, sleepInterval time.Duration) (txResp *txtypes.GetTxResponse, err error) {
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

		txResp, err = c.txClient.GetTx(ctx, &txtypes.GetTxRequest{Hash: txHash})
		if err != nil {
			if strings.Contains(err.Error(), "tx not found") {
				continue
			}

			return txResp, fmt.Errorf("GetTx: %w", err)
		}

		if txResp == nil {
			continue
		}

		if txResp.TxResponse.Code != 0 {
			return txResp, fmt.Errorf("non-zero code: %d", txResp.TxResponse.Code)
		}

		break
	}

	if err != nil {
		return txResp, fmt.Errorf("GetTx: %w", err)
	}

	if txResp == nil || len(txResp.TxResponse.TxHash) == 0 {
		return txResp, errors.New("fail to get tx after retries")
	}

	return
}

// HandleTxsByHeight retrieves contract transaction by height and process via callback.
func (c *Client) HandleTxsByHeight(ctx context.Context, contractAddress string, heightFrom, heightTo int64, acknowledge func(ctx context.Context, msg []abci.Event) error) error {
	tendermintNode, err := c.clientCtx.GetNode()
	if err != nil {
		return fmt.Errorf("clientCtx.GetNode: %w", err)
	}

	txsCount := 100
	from := heightFrom
	to := min(heightFrom+rangeSize, heightTo)

	for {
		page := 1
		query := fmt.Sprintf(searchByHeightQuery, from, to, contractAddress)

		for {
			resp, err := tendermintNode.TxSearch(ctx, query, true, &page, &txsCount, "asc")
			if err != nil {
				return fmt.Errorf("tendermintNode.TxSearch: %w", err)
			}

			for i := range resp.Txs {
				// Create tx_hash event
				txHashEvent := abci.Event{
					Type: "tx",
					Attributes: []abci.EventAttribute{
						{
							Key:   []byte("hash"),
							Value: []byte(hex.EncodeToString(resp.Txs[i].Hash)),
						},
					},
				}

				// Create tx_height event
				txHeightEvent := abci.Event{
					Type: "tx",
					Attributes: []abci.EventAttribute{
						{
							Key:   []byte("height"),
							Value: []byte(strconv.FormatInt(resp.Txs[i].Height, 10)),
						},
					},
				}

				resp.Txs[i].TxResult.Events = append(resp.Txs[i].TxResult.Events, []abci.Event{txHeightEvent, txHashEvent}...)
				err = acknowledge(ctx, resp.Txs[i].TxResult.Events)
				if err != nil {
					return err
				}
			}

			if len(resp.Txs) < 100 {
				if to == heightTo {
					return nil
				}
				from = to
				to = min(from+rangeSize, heightTo)
				break
			}

			page++
		}
	}
}
