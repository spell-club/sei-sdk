package sdk

type SubscribeMessage struct {
	Result struct {
		Events struct {
			WasmContractAddress []string `json:"wasm._contract_address"`
			WasmReferralAddr    []string `json:"wasm.referral_addr"`
			WasmReferralAmount  []string `json:"wasm.referral_amount"`
			WasmAddress         []string `json:"wasm.address"`
			TxHeight            []string `json:"tx.height"`
			TxHash              []string `json:"tx.hash"`
		} `json:"events"`
	} `json:"result"`
}
