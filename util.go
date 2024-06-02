package sdk

import "github.com/cosmos/cosmos-sdk/types/bech32"

func IsValidBlockchainAddress(address string) bool {
	hrp, _, err := bech32.DecodeAndConvert(address)
	if err != nil || hrp != Bech32PrefixAccAddr {
		return false
	}

	return true
}

func ConvertAddr(address, hrp string) (string, error) {
	_, addrBytes, err := bech32.DecodeAndConvert(address)
	if err != nil {
		return "", err
	}

	return bech32.ConvertAndEncode(hrp, addrBytes)
}
