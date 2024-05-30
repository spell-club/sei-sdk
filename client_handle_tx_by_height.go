package sdk

import (
	"context"
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"
)

const searchByHeightQuery = `tx.height>%d AND tx.height<%d AND wasm._contract_address CONTAINS '%s' AND wasm.action='execute_claim'`

func (c *Client) HandleTxsByHeight(ctx context.Context, contractAddress string, heightFrom, heightTo int64, acknowledge func(ctx context.Context, msg []abci.Event) error) error {
	query := fmt.Sprintf(searchByHeightQuery, heightFrom, heightTo, contractAddress)
	tendermintNode, err := c.sign.ctx.GetNode()
	if err != nil {
		return err
	}

	page := 1
	txsCount := 100

	for {
		resp, err := tendermintNode.TxSearch(ctx, query, true, &page, &txsCount, "desc")
		if err != nil {
			return fmt.Errorf("tendermintNode.TxSearch: %w", err)
		}

		for i := range resp.Txs {
			err = acknowledge(ctx, resp.Txs[i].TxResult.Events)
			if err != nil {
				return err
			}
		}

		if len(resp.Txs) < 100 {
			break
		}

		page++
	}

	return nil
}
