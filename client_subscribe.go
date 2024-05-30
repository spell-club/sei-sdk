package sdk

import (
	"context"
	"fmt"

	abci "github.com/tendermint/tendermint/abci/types"
)

var subscribeQuery = "tm.event='Tx' AND wasm._contract_address CONTAINS '%s' AND wasm.action='execute_claim'"

func (c *Client) Subscribe(ctx context.Context, contractAddress string, acknowledge func(ctx context.Context, msg []abci.Event) error) error {
	// Create context to control explicitly ws subscription
	wsCtx, cancelWsCtx := context.WithCancel(context.Background())
	defer cancelWsCtx()

	tendermintNode, err := c.sign.ctx.GetNode()
	if err != nil {
		return fmt.Errorf("ctx.GetNode(): %v", err)
	}

	err = tendermintNode.Start(wsCtx)
	if err != nil {
		return fmt.Errorf("tendermintNode.Start(): %v", err)
	}

	// Subscriber field will be rewritten by tendermint using IP address
	// Also ctx does not control the lifetime of the channel, so we need to stop reading from channel by ourselves
	eventsChan, err := tendermintNode.Subscribe(wsCtx, "", fmt.Sprintf(subscribeQuery, contractAddress))
	if err != nil {
		return fmt.Errorf("tendermintNode.Subscribe: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			err = tendermintNode.UnsubscribeAll(wsCtx, "")
			if err != nil {
				return fmt.Errorf("tendermintNode.UnsubscribeAll: %v", err)
			}

			return nil
		case event := <-eventsChan:
			err = acknowledge(ctx, event.Events)
			if err != nil {
				return fmt.Errorf("acknowledge: %v", err)
			}
		}
	}
}
