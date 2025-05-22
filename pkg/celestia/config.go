package celestia

// Configuration for connecting to Celestia DA
type DAConfig struct {
	Rpc         string
	NamespaceId string
	AuthToken   string
	MaxTxSize   int
}
