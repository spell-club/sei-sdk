package sdk

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	log "github.com/sirupsen/logrus"
)

func TestClient_SendTx(t *testing.T) {
	cfg := Config{
		Network:  "testnet",
		TxMode:   "single",
		GRPCHost: "grpc.atlantic-2.seinetwork.io:443",
		RPCHost:  "https://rpc.atlantic-2.seinetwork.io",

		SignerName:     "user",
		SignerMnemonic: "x",
	}

	logger := log.WithFields(log.Fields{"module": "api"})

	client, err := NewClient(cfg, logger)
	assert.NilError(t, err)

	type ClaimMsg struct {
		Claim struct {
			Address         string `json:"address"`
			ReferralAddress string `json:"referral_address"`
		} `json:"claim"`
	}

	var msg ClaimMsg
	msg.Claim.Address = sdktypes.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	msg.Claim.ReferralAddress = "sei1zydvsxz5x40tcvywdqe8rahj59xyjwtztfjlq6"

	marshalledMsg, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal: %s", err)
	}

	hash, err := client.Execute("sei154p8wkvvgvkrm849ahnw9xwx6v4yj8c9wmfwc83x4u6shcmdyq9qavegg7", []string{string(marshalledMsg)})
	assert.NilError(t, err)

	log.Printf("\nhash: %s", hash)
}
