package etherman

import "errors"

var (
	ErrTokenNotCreated error = errors.New("token does not exist on the network")
)
