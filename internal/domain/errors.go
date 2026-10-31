package domain

import (
	"errors"
	"fmt"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrOutOfScope = errors.New("outside every authorized scope")
	ErrDuplicate  = errors.New("already exists")
	ErrNotFound   = errors.New("not found")
)

type validationError struct {
	message string
}

func (e validationError) Error() string {
	return e.message
}

func (e validationError) Is(target error) bool {
	return target == ErrValidation
}

func validationErrorf(format string, args ...any) error {
	return validationError{message: fmt.Sprintf(format, args...)}
}
