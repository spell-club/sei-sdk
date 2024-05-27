package sdk

import (
	"errors"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdkt "github.com/cosmos/cosmos-sdk/types"
)

func (c *Client) Instantiate(codeID uint64, label, instantiateMsg string, funds []sdkt.Coin) (string, error) {
	if instantiateMsg == "" {
		return "", errors.New("message code is empty")
	}

	if label == "" {
		return "", errors.New("label is empty")
	}

	if len(funds) == 0 {
		return "", errors.New("funds are empty")
	}

	message := &wasmtypes.MsgInstantiateContract{
		Sender: c.sign.sender,
		Admin:  c.sign.sender,
		Label:  label,
		CodeID: codeID,
		Msg:    []byte(instantiateMsg),
		Funds:  funds,
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
