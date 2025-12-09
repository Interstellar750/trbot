package flaterr

import (
	"errors"
	"fmt"
	"strings"
)

type MultErr struct {
	Errors []error
}

// Error returns the errors string to implement the error interface
func (e *MultErr) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, err := range e.Errors {
		sb.WriteString(err.Error())
		sb.WriteRune('\n')
	}
	return sb.String()
}

// Add add errors to MultErr
func (e *MultErr) Add(errs ...error) *MultErr {
	for _, err := range errs {
		if err != nil {
			e.Errors = append(e.Errors, err)
		}
	}
	return e
}

// Addf add formatted error by use fmt.Errorf()
func (e *MultErr) Addf(format string, a ...any) *MultErr {
	e.Errors = append(e.Errors, fmt.Errorf(format, a...))
	return e
}

// Addt add template error by use fmt.Errorf()
func (e *MultErr) Addt(msg Msg, content string, err error) *MultErr {
	e.Errors = append(e.Errors, fmt.Errorf(msg.Fmt(), content, err))
	return e
}

// Flat return multiple errors by use errors.Join()
func (e *MultErr) Flat() error {
	if len(e.Errors) == 1 {
		return e.Errors[0]
	}

	if len(e.Errors) > 1 {
		return errors.Join(e.Errors...)
	}

	return nil}
