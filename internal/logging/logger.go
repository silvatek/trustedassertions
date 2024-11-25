package logging

import (
	"context"
	"fmt"
	"io"
	"os"
)

type Logger struct {
	Name       string
	Level      int
	Structured bool
	Writer     io.Writer
}

var loggers map[string]Logger = nil

func GetLogger(name string) Logger {
	if loggers == nil {
		loggers = make(map[string]Logger, 0)
	}
	log, found := loggers[name]
	if !found {
		log = Logger{Name: name, Level: DEBUG, Structured: structuredLogging(), Writer: LogWriter}
		loggers[name] = log
	}
	return log
}

func structuredLogging() bool {
	return (os.Getenv("LOG_TYPE") == "structured") || (os.Getenv("GCLOUD_PROJECT") != "")
}

func (log Logger) Print(args ...any) {
	text := ""
	for _, arg := range args {
		if len(text) > 0 {
			text += " "
		}
		text += fmt.Sprintf("%v", arg)
	}
	log.Info(text)
}

func (log Logger) Println(args ...any) {
	log.Print(args...)
}

func (log Logger) Printf(template string, args ...any) {
	log.Infof(template, args...)
}

func (log Logger) DebugfX(ctx context.Context, text string, args ...interface{}) {
	if log.Level >= DEBUG {
		WriteNamedLog(ctx, log.Writer, log.Structured, log.Name, "DEBUG", text, args...)
	}
}

func (log Logger) Debugf(text string, args ...interface{}) {
	log.DebugfX(context.Background(), text, args...)
}

func (log Logger) Info(text string) {
	log.InfofX(context.Background(), text)
}

func (log Logger) Infof(text string, args ...interface{}) {
	log.InfofX(context.Background(), text, args...)
}

func (log Logger) InfofX(ctx context.Context, text string, args ...interface{}) {
	if log.Level <= INFO {
		WriteNamedLog(ctx, log.Writer, log.Structured, log.Name, "INFO ", text, args...)
	}
}

func (log Logger) ErrorfX(ctx context.Context, text string, args ...interface{}) {
	if log.Level <= ERROR {
		WriteNamedLog(ctx, log.Writer, log.Structured, log.Name, "ERROR", text, args...)
	}
}

func (log Logger) Errorf(text string, args ...interface{}) {
	log.ErrorfX(context.Background(), text, args...)
}
