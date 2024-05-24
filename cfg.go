package sei_sdk

type Config struct {
	Network    string
	Contract   string
	TxMode     string
	ChainID    string
	RPCAddress string // "grpc.atlantic-2.seinetwork.io:443"
	NodeURI    string // "https://rpc.atlantic-2.seinetwork.io"
}
