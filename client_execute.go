package sdk

import (
	"context"
	"errors"
	"fmt"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

const (
	defaultTimeoutHeight             = 20
	defaultTimeoutHeightSyncInterval = 10 * time.Second
)

func (c *Client) Execute(ctx context.Context, contractAddress, msg string) (string, error) {
	if msg == "" {
		return "", errors.New("message is empty")
	}

	txHash, err := c.asyncBroadcastMsg(ctx, &wasmtypes.MsgExecuteContract{
		Sender:   c.sign.sender,
		Contract: contractAddress,
		Msg:      []byte(msg),
	})
	if err != nil {
		return "", fmt.Errorf("AsyncBroadcastMsg: %s", err)
	}

	return txHash, nil
}
