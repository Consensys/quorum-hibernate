package core

import "errors"

var ErrNodeDown = errors.New("node is not up")

var ErrTxNotPvt = errors.New("not a private tx")
