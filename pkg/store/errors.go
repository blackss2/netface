package store

import (
	"errors"
)

var (
	ErrNotSupportedMethod      = errors.New("not supported method")
	ErrNotSupportedDriver      = errors.New("not supported driver")
	ErrNotExist                = errors.New("not exist")
	ErrNotExistTable           = errors.New("not exist table")
	ErrNotExistIndex           = errors.New("not exist index")
	ErrNotPermitted            = errors.New("not permitted")
	ErrInvalidArgument         = errors.New("invalid argument")
	ErrInvalidConnectionString = errors.New("invalid connection string")
)
