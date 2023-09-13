package config

import "github.com/cockroachdb/errors"

var (
	ErrNotInitial   = errors.New("config is not initialized")
	ErrIgnoreChange = errors.New("ignore change")
	ErrKeyNotFound  = errors.New("key not found")
)
