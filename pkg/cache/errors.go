package cache

import (
	"errors"
)

var (
	ErrNotSupportedMethod      = errors.New("not supported method")
	ErrNotSupportedDriver      = errors.New("not supported driver")
	ErrNotExist                = errors.New("not exist")
	ErrNotPermitted            = errors.New("not permitted")
	ErrInvalidArgument         = errors.New("invalid argument")
	ErrInvalidConnectionString = errors.New("invalid connection string")
	ErrInvalidHost             = errors.New("invalid host")
	ErrInvalidCache            = errors.New("invalid cache")
)
