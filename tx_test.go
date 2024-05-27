package seisdk

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/sirupsen/logrus"
)

func TestClient_SendTx(t *testing.T) {
	cfg := Config{
		Network:  "testnet",
		Contract: "sei154p8wkvvgvkrm849ahnw9xwx6v4yj8c9wmfwc83x4u6shcmdyq9qavegg7",
		TxMode:   "single",
		ChainID:  "atlantic-2",
		GRPCHost: "grpc.atlantic-2.seinetwork.io:443",
		RPCHost:  "https://rpc.atlantic-2.seinetwork.io",
	}

	logger := log.WithFields(log.Fields{"module": "api"})

	client, err := NewClient(cfg, logger)
	assert.NilError(t, err)

	type ClaimMsg struct {
		Claim struct {
			Address string `json:"address"`
		} `json:"claim"`
	}

	var msg ClaimMsg
	msg.Claim.Address = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	marshalledMsg, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %s", err)
	}

	client.AddSigner("user", "hurt monster burger grocery drill afraid muffin rubber grid fuel clinic fuel")

	hash, err := client.Execute("", []string{string(marshalledMsg)})
	assert.NilError(t, err)

	log.Printf("\nhash: %s", hash)
}
