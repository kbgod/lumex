package zerolog

import (
	"github.com/kbgod/lumex/log"
	"github.com/rs/zerolog"
)

var _ log.Logger = (*LogAdapter)(nil)

type LogAdapter struct {
	log *zerolog.Logger
}

func NewAdapter(log *zerolog.Logger) log.Logger {
	return &LogAdapter{
		log: log,
	}
}

func (a *LogAdapter) Error(err error, message string, fields map[string]any) {
	a.log.Error().Err(err).Fields(fields).Msg(message)
}

func (a *LogAdapter) Info(message string, fields map[string]any) {
	a.log.Info().Fields(fields).Msg(message)
}

func (a *LogAdapter) Debug(message string, fields map[string]any) {
	a.log.Debug().Fields(fields).Msg(message)
}
