package seisdk

import (
	"errors"
	"fmt"
	"strings"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	defaultTimeoutHeight             = 20
	defaultTimeoutHeightSyncInterval = 10 * time.Second
	msgBatchLen                      = 50
)

func (c *Client) Execute(contractAddress string, msgs []string) (string, error) {
	if len(msgs) == 0 {
		return "", errors.New("message is empty")
	}
	if len(msgs) > msgBatchLen {
		return "", errors.New("too many messages")
	}

	txResult, err := c.asyncBroadcastMsg(Map(msgs, func(d string) sdk.Msg {
		return &wasmtypes.MsgExecuteContract{
			Sender:   c.sign.sender,
			Contract: contractAddress,
			Msg:      []byte(d),
		}
	})...)
	if err != nil {
		if strings.Contains(err.Error(), "is greater than max gas") && len(msgs) > 2 {
			var txHashR string

			for _, chunk := range Chunk(msgs, len(msgs)/2+1) {
				txHashR, err = c.Execute(contractAddress, chunk)
				if err != nil {
					return "", fmt.Errorf("Execute recursive call: %s", err)
				}
			}

			return txHashR, nil
		}

		return "", fmt.Errorf("AsyncBroadcastMsg: %s", err)
	}

	if txResult == nil || txResult.GetTxResponse() == nil {
		return "", fmt.Errorf("txResult is nil: %v", txResult)
	}

	return txResult.GetTxResponse().TxHash, nil
}
