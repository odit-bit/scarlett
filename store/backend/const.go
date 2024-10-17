package backend

import "errors"

var ErrItemExpired = errors.New("item is expired")
var ErrInvalidCmd = errors.New("invalid cmd")
var ErrKeyNotFound = errors.New("key not found")

var (
	_bolt_kv = []byte("bolt_kv")
)
