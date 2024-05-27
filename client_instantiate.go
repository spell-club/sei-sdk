package seisdk

import (
	"errors"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func (c *Client) Instantiate(address string, code string) (string, error) {
	if code == "" {
		return "", errors.New("message code is empty")
	}

	message := &wasmtypes.MsgInstantiateContract{
		Sender: address,
		Msg:    []byte(code),
	}

	txResult, err := c.asyncBroadcastMsg(message)
	if err != nil {
		return "", fmt.Errorf("AsyncBroadcastMsg: %s", err)
	}

	if txResult == nil || txResult.GetTxResponse() == nil {
		return "", fmt.Errorf("txResult is nil: %v", txResult)
	}

	return txResult.GetTxResponse().TxHash, nil
}
