package botstate

import "errors"

var ErrEventRejected = errors.New("event rejected")
var ErrSessionNotFound = errors.New("not found")
var ErrStateNotFound = errors.New("not found")
