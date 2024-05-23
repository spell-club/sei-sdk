package sei_sdk

type ClientConfig struct {
	Network    string
	Contract   string
	Key        string
	TxMode     string
	ChainID    string
	RPCAddress string // "grpc.atlantic-2.seinetwork.io:443"
	NodeURI    string // "https://rpc.atlantic-2.seinetwork.io"
	KeyringUID string // "user"
}
