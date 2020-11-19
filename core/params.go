package core

import (
	"time"
)

const (
	HttpClientRequestTimeout       = 10 * time.Second
	HttpClientRequestDialerTimeout = 10 * time.Second
	TLSHandshakeTimeout            = 10 * time.Second
	Qnm2QnmValidationRetryLimit    = 3
)
