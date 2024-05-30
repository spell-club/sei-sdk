package sdk

import (
	"context"
	"fmt"
	"sync"

	abci "github.com/tendermint/tendermint/abci/types"
)

var subscribeQuery = "tm.event='Tx' AND wasm._contract_address CONTAINS '%s' AND wasm.action='execute_claim'"

func (c *Client) Subscribe(ctx context.Context, contractAddress string, acknowledge func(ctx context.Context, msg []abci.Event) error) error {
	if c.wss.host == "" {
		return fmt.Errorf("websocket host wasn't provided in config")
	}

	tendermintNode, err := c.sign.ctx.GetNode()
	if err != nil {
		return fmt.Errorf("ctx.GetNode(): %v", err)
	}

	eventsChan, err := tendermintNode.Subscribe(ctx, "", fmt.Sprintf(subscribeQuery, contractAddress))
	if err != nil {
		return fmt.Errorf("tendermintNode.Subscribe: %v", err)
	}

	wg := &sync.WaitGroup{}
	go func() {
		wg.Add(1)
		defer wg.Done()
		for event := range eventsChan {
			err = acknowledge(ctx, event.Events)
			if err != nil {
				c.logger.Logger.Errorf("conn.ReadMessage: %s", err)

				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		err = tendermintNode.UnsubscribeAll(ctx, "")
		if err != nil {
			return fmt.Errorf("tendermintNode.UnsubscribeAll: %v", err)
		}
		wg.Wait()

		return nil
	}
}
