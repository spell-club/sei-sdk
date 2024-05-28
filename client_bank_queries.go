package sdk

import (
	"context"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const DefaultDenom = "usei"

func (c *Client) GetBankBalance(ctx context.Context, address, denom string) (*banktypes.QueryBalanceResponse, error) {
	req := &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	}
	return c.bankQueryClient.Balance(ctx, req)
}
