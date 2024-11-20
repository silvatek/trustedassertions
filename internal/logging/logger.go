package logging

import (
	"context"
)

type Logger struct {
	Name  string
	Level int
}

var loggers = make(map[string]Logger, 0)

func GetLogger(name string) Logger {
	log, ok := loggers[name]
	if !ok {
		log = Logger{Name: name, Level: INFO}
		loggers[name] = log
	}
	return log
}

func (log Logger) DebugfX(ctx context.Context, text string, args ...interface{}) {
	if log.Level >= DEBUG {
		WriteNamedLog(ctx, log.Name, "DEBUG", text, args...)
	}
}

func (log Logger) InfofX(ctx context.Context, text string, args ...interface{}) {
	if log.Level >= INFO {
		WriteNamedLog(ctx, log.Name, "INFO ", text, args...)
	}
}

func (log Logger) ErrorfX(ctx context.Context, text string, args ...interface{}) {
	if log.Level >= ERROR {
		WriteNamedLog(ctx, log.Name, "ERROR", text, args...)
	}
}
