package sei_sdk

type ClientConfig struct {
	Network    string `required:"true" split_words:"true"`
	Contract   string `required:"true" split_words:"true"`
	Key        string `required:"true" split_words:"true"`
	TxMode     string `required:"true" split_words:"true"`
	RPCAddress string `required:"true" split_words:"true"` // "grpc.atlantic-2.seinetwork.io:443"
	NodeURI    string `required:"true" split_words:"true"` // "https://rpc.atlantic-2.seinetwork.io"
	KeyringUID string `required:"true" split_words:"true"` // "user"
}
