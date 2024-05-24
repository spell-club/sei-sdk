package sei_sdk

import (
	"encoding/json"
	"log"
	"testing"

	"gotest.tools/assert"
)

func TestClient_SendTx(t *testing.T) {
	cfg := Config{
		Network:    "testnet",
		Contract:   "sei154p8wkvvgvkrm849ahnw9xwx6v4yj8c9wmfwc83x4u6shcmdyq9qavegg7",
		TxMode:     "single",
		ChainID:    "atlantic-2",
		RPCAddress: "grpc.atlantic-2.seinetwork.io:443",
		NodeURI:    "https://rpc.atlantic-2.seinetwork.io",
	}

	client, err := NewClient(cfg)
	assert.NilError(t, err)

	type ClaimMsg struct {
		Claim struct {
			Address string `json:"address"`
		} `json:"claim"`
	}

	var msg ClaimMsg
	msg.Claim.Address = "x"

	marshalledMsg, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %s", err)
	}

	client.AddSigner("user", "hurt monster burger grocery drill afraid muffin rubber grid fuel clinic fuel")

	hash, err := client.SendTx([]string{string(marshalledMsg)})
	assert.NilError(t, err)

	log.Printf("\nhash: %s", hash)
}
