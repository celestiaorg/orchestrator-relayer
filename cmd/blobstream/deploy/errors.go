package deploy

import "errors"

var (
	ErrUnmarshallValset = errors.New("couldn't unmarshall valset")
	ErrNotFound         = errors.New("not found")
)
