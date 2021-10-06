package botstate

import "errors"

var (
	ErrEventRejected   = errors.New("event rejected")
	ErrSessionNotFound = errors.New("not found")
	ErrStateNotFound   = errors.New("not found")
)
