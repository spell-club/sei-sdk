package sei_sdk

type Config struct {
	Network  string
	Contract string
	TxMode   string
	ChainID  string
	GRPCHost string // "grpc.atlantic-2.seinetwork.io:443"
	RPCHost  string // "https://rpc.atlantic-2.seinetwork.io"
}
