package sdk

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

func (c *Client) GetTxByHash(ctx context.Context, txHash string) (*types.TxResponse, error) {
	txResp, err := c.txClient.GetTx(ctx, &tx.GetTxRequest{Hash: txHash})
	if err != nil {
		return nil, fmt.Errorf("txClient.GetTx: %w", err)
	}

	return txResp.TxResponse, nil
}
