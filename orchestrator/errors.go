package orchestrator

import "errors"

var (
	ErrEmptyPeersTable = errors.New("empty peers table")
	ErrSignalChanNotif = errors.New("signal channel sent notification to stop")
)
