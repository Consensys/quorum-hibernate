package privatetx

// TxHandler is an interface to process private transactions.
// It should be implementd for quorum and besu as they have differences
// in message formats in handling private transactions.
type TxHandler interface {
	// IsPrivateTx will take msg as input and return an array of public keys of participants if
	// the msg is a private transaction.
	IsPrivateTx(msg []byte) ([]string, error)
}
