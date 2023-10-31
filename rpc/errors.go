package rpc

import "errors"

var (
	ErrCouldntReachSpecifiedHeight = errors.New("couldn't reach specified height")
	ErrNotFound                    = errors.New("not found")
)
