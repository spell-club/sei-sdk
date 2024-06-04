package sdk

import (
	"context"
	"fmt"
)

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
