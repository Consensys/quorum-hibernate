package core

import (
	"time"
)

const (
	HttpClientRequestTimeout       = 10 * time.Second
	HttpClientRequestDialerTimeout = 10 * time.Second
	TLSHandshakeTimeout            = 10 * time.Second
	Peer2PeerValidationRetryLimit  = 3
)
