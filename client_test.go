package sdk

import (
	"context"
	"testing"

	"gotest.tools/assert"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

const (
	testKeyName     = "name"
	testKeyMnemonic = "mnemonic"
	TestnetGRPCHost = "grpc.atlantic-2.seinetwork.io:443"
	TestnetRPCHost  = "https://rpc.atlantic-2.seinetwork.io"
)

func TestClient_SendTx(t *testing.T) {
	cfg := Config{
		ChainID:  ChainIDTestnet,
		GRPCHost: TestnetGRPCHost,
		RPCHost:  TestnetRPCHost,
	}

	client, err := NewClient(cfg)
	assert.NilError(t, err)
	_, err = client.AddSigner(testKeyName, testKeyMnemonic)
	assert.NilError(t, err)

	type ClaimMsg struct {
		Claim struct {
			Address string `json:"address"`
		} `json:"claim"`
	}

	var msg ClaimMsg
	msg.Claim.Address = sdktypes.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	hash, err := client.ExecuteJson(context.Background(), testKeyName, "sei154p8wkvvgvkrm849ahnw9xwx6v4yj8c9wmfwc83x4u6shcmdyq9qavegg7", msg)
	assert.NilError(t, err)

	t.Logf("hash: %s", hash.GetTxResponse().TxHash)
}

func TestClient_Query(t *testing.T) {
	cfg := Config{
		ChainID:  ChainIDTestnet,
		GRPCHost: TestnetGRPCHost,
		RPCHost:  TestnetRPCHost,
	}

	client, err := NewClient(cfg)
	assert.NilError(t, err)

	balance, err := client.GetBankBalance(context.Background(), "sei1mce4kk5a0spf2nlg9z6a7ryncz3qksgg4wr7fs", DefaultDenom)
	assert.NilError(t, err)

	t.Logf("balance: %s", balance.Balance.String())
}

func TestClient_Acc(t *testing.T) {
	cfg := Config{
		ChainID:  ChainIDTestnet,
		GRPCHost: TestnetGRPCHost,
		RPCHost:  TestnetRPCHost,
	}

	client, err := NewClient(cfg)
	assert.NilError(t, err)
	_, err = client.AddSigner(testKeyName, testKeyMnemonic)
	assert.NilError(t, err)

	sgn, err := client.getSigner(testKeyName)
	assert.NilError(t, err)

	num, seq, err := client.clientCtx.AccountRetriever.GetAccountNumberSequence(client.clientCtx, sgn.address.Bytes())
	assert.NilError(t, err)

	t.Logf("num: %d seq: %d", num, seq)
}
