package sdk

import (
	"context"
	"errors"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

func (c *Client) Instantiate(ctx context.Context, codeID uint64, label, instantiateMsg string, funds []sdktypes.Coin) (string, error) {
	if instantiateMsg == "" {
		return "", errors.New("message code is empty")
	}
	if label == "" {
		return "", errors.New("label is empty")
	}

	message := &wasmtypes.MsgInstantiateContract{
		Sender: c.sign.sender,
		Admin:  c.sign.sender,
		Label:  label,
		CodeID: codeID,
		Msg:    []byte(instantiateMsg),
		Funds:  funds,
	}
	txHash, err := c.asyncBroadcastMsg(ctx, message)
	if err != nil {
		return "", fmt.Errorf("AsyncBroadcastMsg: %s", err)
	}

	return txHash, nil
}
