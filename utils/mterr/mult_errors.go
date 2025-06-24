package mterr

import (
	"errors"
	"fmt"
)

type MultiError struct {
	Errors []error
}

func (e *MultiError) Add(errs ...error) *MultiError {
	for _, err := range errs {
		if err != nil {
			e.Errors = append(e.Errors, err)
		}
	}
	return e
}

// add formatted error by use fmt.Errorf()
func (e *MultiError) Addf(format string, a ...any) *MultiError {
	e.Errors = append(e.Errors, fmt.Errorf(format, a...))
	return e
}

// func (e *MultiError) AddWith(key string, dict *zerolog.Event) string {
// 	dict.
// }

// a string error by use errors.Join()
func (e *MultiError) Flat() error {
	if len(e.Errors) == 0 {
		return nil
	}
	return errors.Join(e.Errors...)
}
