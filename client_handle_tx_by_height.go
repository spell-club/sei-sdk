package sdk

import (
	"context"
	"encoding/hex"
	"fmt"
	abci "github.com/tendermint/tendermint/abci/types"
	"strconv"
)

const (
	searchByHeightQuery = `tx.height>%d AND tx.height<=%d AND wasm._contract_address CONTAINS '%s'`
	rangeSize           = 100_000
)

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
