package task

import (
	"fmt"

	"github.com/rs/zerolog"
)

// FormatKeyValue adds every two args to the event by calling `zerolog.Event().Str()`.
func FormatKeyValue(event *zerolog.Event, args ...any) *zerolog.Event {
	var key, value string
	for index, arg := range args {
		if index % 2 == 0 {
			key = fmt.Sprint(arg)
		} else {
			value = fmt.Sprint(arg)
			event = event.Str(key, value)
		}
	}
	return event
}

// NewZerologWapped wraps `zerolog.Logger` to implements the github.com/reugn/go-quartz/logger interface.
//
// https://pkg.go.dev/github.com/reugn/go-quartz/logger#Logger
func NewZerologWappred(zerolog zerolog.Logger) *ZerologWappred {
	return &ZerologWappred{
		zerolog: zerolog.With().
			Str("package", "go-quartz").
			Logger(),
	}
}

type ZerologWappred struct {
	zerolog zerolog.Logger
}

func (z *ZerologWappred) Trace(msg string, args ...any) {
	if len(args) == 0 {
		z.zerolog.Trace().Msg(msg)
		return
	} else if len(args) % 2 == 0 {
		FormatKeyValue(z.zerolog.Trace(), args...).Msg(msg)
	} else {
		z.zerolog.Trace().Msgf(msg, args...)
	}
}

func (z *ZerologWappred) Debug(msg string, args ...any) {
	if len(args) == 0 {
		z.zerolog.Debug().Msg(msg)
		return
	} else if len(args)%2 == 0 {
		FormatKeyValue(z.zerolog.Debug(), args...).Msg(msg)
	} else {
		z.zerolog.Debug().Msgf(msg, args...)
	}
}

func (z *ZerologWappred) Info(msg string, args ...any) {
	if len(args) == 0 {
		z.zerolog.Info().Msg(msg)
		return
	} else if len(args)%2 == 0 {
		FormatKeyValue(z.zerolog.Info(), args...).Msg(msg)
	} else {
		z.zerolog.Info().Msgf(msg, args...)
	}
}

func (z *ZerologWappred) Warn(msg string, args ...any) {
	if len(args) == 0 {
		z.zerolog.Warn().Msg(msg)
		return
	} else if len(args)%2 == 0 {
		FormatKeyValue(z.zerolog.Warn(), args...).Msg(msg)
	} else {
		z.zerolog.Warn().Msgf(msg, args...)
	}
}

func (z *ZerologWappred) Error(msg string, args ...any) {
	if len(args) == 0 {
		z.zerolog.Error().Msg(msg)
		return
	} else if len(args)%2 == 0 {
		FormatKeyValue(z.zerolog.Error(), args...).Msg(msg)
	} else {
		z.zerolog.Error().Msgf(msg, args...)
	}
}
