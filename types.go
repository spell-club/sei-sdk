package sdk

type SubscribeMessage struct {
	Result struct {
		Events struct {
			TxHeight            []string `json:"tx.height"`
			TxHash              []string `json:"tx.hash"`
			WasmContractAddress []string `json:"wasm._contract_address"`
			WasmAddress         []string `json:"wasm.address"`
			WasmAmount          []string `json:"wasm.amount"`
			WasmReferralAddr    []string `json:"wasm.referral_addr"`
			WasmReferralAmount  []string `json:"wasm.referral_amount"`
		} `json:"events"`
	} `json:"result"`
}
