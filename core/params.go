package core

import (
	"time"
)

const (
	HttpClientRequestTimeout       = 10 * time.Second
	HttpClientRequestDialerTimeout = 10 * time.Second
	TLSHandshakeTimeout            = 10 * time.Second
	Peer2PeerValidationRetryLimit  = 3

	// Message to DAPP / Clients of blockchain
	NodeIsBeingShutdown           = "node is being shutdown, try after sometime"
	NodeIsBeingStarted            = "node is being started, try after sometime"
	SomeParticipantsDown          = "Some participant nodes are down"
	NodeIsNotReadyToAcceptRequest = "node is not ready to accept request"
)
