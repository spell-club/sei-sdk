package sdk

type SubscribeMessage struct {
	Result Result `json:"result"`
}

type Result struct {
	SubscriptionID string `json:"subscription_id"`
	Query          string `json:"query"`
	Data           Data   `json:"data"`
}

type Data struct {
	Value Value `json:"value"`
}

type Value struct {
	TxResult TxResult `json:"TxResult"`
}

type TxResult struct {
	Result ResultDetail `json:"result"`
}

type ResultDetail struct {
	Log string `json:"log"`
}
