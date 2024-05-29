package sdk

import (
	"context"
	"testing"
	"time"

	"gotest.tools/assert"

	log "github.com/sirupsen/logrus"
)

func TestClient_Subscribe(t *testing.T) {
	cfg := Config{
		Network:  "testnet",
		TxMode:   "single",
		GRPCHost: "grpc.atlantic-2.seinetwork.io:443",
		RPCHost:  "https://rpc.atlantic-2.seinetwork.io",
		WSSHost:  "wss://rpc.atlantic-2.seinetwork.io/websocket",

		SignerName:     "user",
		SignerMnemonic: "x",
	}

	logger := log.WithFields(log.Fields{"module": "api"})

	client, err := NewClient(cfg, logger)
	assert.NilError(t, err)

	acknowledge := func(msg SubscribeMessage) error {
		log.Printf("%s", msg.Result.Data.Value.TxResult.Result.Log)
		return nil
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)

	err = client.Subscribe(ctx, "sei154p8wkvvgvkrm849ahnw9xwx6v4yj8c9wmfwc83x4u6shcmdyq9qavegg7", acknowledge)
	assert.NilError(t, err)
}
