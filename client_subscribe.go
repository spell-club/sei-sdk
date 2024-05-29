package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const reconnectDelay = 5 * time.Second

var subscribeMessage = `{
	"jsonrpc": "2.0",
	"method": "subscribe",
	"id": 0,
	"params": {
		"query": "tm.event='Tx' AND wasm._contract_address CONTAINS '%s' AND wasm.action='execute_claim'"
	}
}`

var unsubscribeMessage = `{
	"jsonrpc": "2.0",
	"method": "unsubscribe",
	"id": 0,
}`

func (c *Client) Subscribe(ctx context.Context, contractAddress string, acknowledge func(ctx context.Context, msg SubscribeMessage) error) error {
	for {
		conn, _, err := websocket.DefaultDialer.Dial(c.wss.host, nil)
		if err != nil {
			return fmt.Errorf("DefaultDialer.Dial: %w", err)
		}

		err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(subscribeMessage, contractAddress)))
		if err != nil {
			return fmt.Errorf("conn.WriteMessage: %w", err)
		}

		done := make(chan struct{})
		go func() {
			for {
				var message []byte
				_, message, err = conn.ReadMessage()
				if err != nil {
					c.logger.Logger.Errorf("conn.ReadMessage: %s", err)

					return
				}

				resp := SubscribeMessage{}
				err = json.NewDecoder(bytes.NewReader(message)).Decode(&resp)
				if err != nil {
					c.logger.Logger.Errorf("Decode: %s", err)

					return
				}

				err = acknowledge(ctx, resp)
				if err != nil {
					c.logger.Logger.Errorf("acknowledge: %s", err)

					return
				}
			}
		}()

		select {
		case <-done:
			c.logger.Warnf("Connection lost, attempting to reconnect...")
			err = conn.Close()
			if err != nil {
				return fmt.Errorf("conn.Close: %w", err)
			}
			time.Sleep(reconnectDelay)
		case <-ctx.Done():
			c.logger.Warnf("Interrupt received, closing connection...")
			err = conn.WriteMessage(websocket.TextMessage, []byte(unsubscribeMessage))
			if err != nil {
				return fmt.Errorf("conn.WriteMessage: %w", err)
			}
			err = conn.Close()
			if err != nil {
				return fmt.Errorf("conn.Close: %w", err)
			}

			return nil
		}
	}
}
