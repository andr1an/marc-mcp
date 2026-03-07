package tools

import "errors"

var (
	ErrToolNotFound    = errors.New("tool not found")
	ErrInvalidArgument = errors.New("invalid argument")
)
