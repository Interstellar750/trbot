package flat_err

import (
	"errors"
	"fmt"
)

type Errors struct {
	Errors []error
}

func (e *Errors) Add(errs ...error) *Errors {
	for _, err := range errs {
		if err != nil {
			e.Errors = append(e.Errors, err)
		}
	}
	return e
}

// add formatted error by use fmt.Errorf()
func (e *Errors) Addf(format string, a ...any) *Errors {
	e.Errors = append(e.Errors, fmt.Errorf(format, a...))
	return e
}

// func (e *MultiError) AddWith(key string, dict *zerolog.Event) string {
// 	dict.
// }

// a string error by use errors.Join()
func (e *Errors) Flat() error {
	if len(e.Errors) == 0 {
		return nil
	}
	return errors.Join(e.Errors...)
}
