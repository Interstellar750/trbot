package flate

import (
	"errors"
	"fmt"
)

type MultErr struct {
	Errors []error
}

// add error to MultErr
func (e *MultErr) Add(errs ...error) *MultErr {
	for _, err := range errs {
		if err != nil {
			e.Errors = append(e.Errors, err)
		}
	}
	return e
}

// add formatted error by use fmt.Errorf()
func (e *MultErr) Addf(format string, a ...any) *MultErr {
	e.Errors = append(e.Errors, fmt.Errorf(format, a...))
	return e
}

// a string error by use errors.Join()
func (e *MultErr) Flat() error {
	if len(e.Errors) == 0 {
		return nil
	}
	return errors.Join(e.Errors...)
}
