package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"silvatek.uk/trustedassertions/internal/appcontext"
)

type LogEntry struct {
	Severity    string            `json:"severity"`
	Timestamp   time.Time         `json:"timestamp"`
	Message     interface{}       `json:"message,omitempty"`
	TextPayload interface{}       `json:"textPayload,omitempty"`
	Labels      map[string]string `json:"logging.googleapis.com/labels,omitempty"`
	TraceID     string            `json:"logging.googleapis.com/trace,omitempty"`
	SpanID      string            `json:"logging.googleapis.com/spanId,omitempty"`
	HttpRequest HttpRequestLog    `json:"httpRequest,omitempty"`
}

type HttpRequestLog struct {
	RequestMethod string `json:"requestMethod,omitempty"`
	RequestUrl    string `json:"requestUrl,omitempty"`
}

var LogWriter io.Writer = os.Stderr
var StructureLogs bool
var encoder *json.Encoder

var exitHandler = os.Exit

func Print(text string) {
	Printf(text)
}

func Printf(text string, args ...interface{}) {
	Infof(text, args...)
}

func Debug(text string) {
	Debugf(text)
}

func Debugf(text string, args ...interface{}) {
	DebugfX(context.Background(), text, args...)
}

func DebugfX(ctx context.Context, text string, args ...interface{}) {
	WriteLog(ctx, "DEBUG", text, args...)
}

func Info(text string) {
	Infof(text)
}

func Infof(text string, args ...interface{}) {
	InfofX(context.Background(), text, args...)
}

func InfofX(ctx context.Context, text string, args ...interface{}) {
	WriteLog(ctx, "INFO ", text, args...)
}

func Errorf(text string, args ...interface{}) {
	ErrorfX(context.Background(), text, args...)
}

func ErrorfX(ctx context.Context, text string, args ...interface{}) {
	WriteLog(ctx, "ERROR", text, args...)
}

func WriteLog(ctx context.Context, level string, template string, args ...interface{}) {
	if StructureLogs {
		if encoder == nil {
			encoder = json.NewEncoder(LogWriter)
		}
		entry := LogEntry{
			Severity:  strings.TrimSpace(level),
			Timestamp: time.Now(),
			Message:   fmt.Sprintf(template, args...),
		}

		labels := map[string]string{
			"appname": "trustedassertions",
		}

		data, _ := appcontext.ContextData(ctx)
		if data.ReqPath != "" {
			labels["reqPath"] = data.ReqPath
		}
		if data.Dummy != "" {
			labels["dummy"] = data.Dummy
		}

		entry.Labels = labels

		encoder.Encode(entry)

	} else {
		fmt.Fprintf(LogWriter, level+" "+template+"\n", args...)
	}
}

func Fatal(err error) {
	Errorf("Fatal error: %v", err)
	exitHandler(1)
}
