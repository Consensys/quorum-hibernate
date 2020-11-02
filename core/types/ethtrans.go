package types

type TransData struct {
	Data       string   `json:"data"`
	From       string   `json:"from"`
	To         string   `json:"to,omitempty"`
	PrivateFor []string `json:"privateFor,omitempty"`
}

type EthTransaction struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      int         `json:"id"`
	Method  string      `json:"method"`
	Params  []TransData `json:"params"`
}
