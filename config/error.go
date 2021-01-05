package config

import (
	"errors"
	"fmt"
)

var (
	isEmptyErr              = errors.New("is empty")
	isNotGreaterThanZeroErr = errors.New("must be > 0")
)

type fieldErr struct {
	field string
	cause error
}

type arrFieldErr struct {
	field string
	i     int
	cause error
}

func newFieldErr(field string, cause error) error {
	return &fieldErr{
		field: field,
		cause: cause,
	}
}

func newArrFieldErr(field string, i int, cause error) error {
	return &arrFieldErr{
		field: field,
		cause: cause,
		i:     i,
	}
}

func (e *fieldErr) Error() string {
	switch e.cause.(type) {
	case *fieldErr, *arrFieldErr:
		return fmt.Sprintf("%v.%v", e.field, e.cause)
	default:
		return fmt.Sprintf("%v %v", e.field, e.cause)
	}
}

func (e *arrFieldErr) Error() string {
	switch e.cause.(type) {
	case *fieldErr, *arrFieldErr:
		return fmt.Sprintf("%v[%v].%v", e.field, e.i, e.cause)
	default:
		return fmt.Sprintf("%v[%v] %v", e.field, e.i, e.cause)
	}
}
