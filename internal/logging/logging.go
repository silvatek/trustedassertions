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
	Sampled     bool              `json:"logging.googleapis.com/traceSampled,omitempty"`
	HttpRequest HttpRequestLog    `json:"httpRequest,omitempty"`
}

type HttpRequestLog struct {
	RequestMethod string `json:"requestMethod,omitempty"`
	RequestUrl    string `json:"requestUrl,omitempty"`
}

const TRACE = 1
const DEBUG = 2
const INFO = 3
const WARN = 4
const ERROR = 5
const FATAL = 6

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
	WriteNamedLog(ctx, LogWriter, StructureLogs, "", level, template, args...)
}

func WriteNamedLog(ctx context.Context, output io.Writer, structured bool, name string, level string, template string, args ...interface{}) {
	if structured {
		encoder = json.NewEncoder(output)
		entry := makeLogEntry(ctx, level, name, template, args...)
		encoder.Encode(entry)
	} else {
		entry := simpleLogMessage(name, level, template, args...)
		fmt.Fprintln(output, entry)
	}
}

func simpleLogMessage(name string, level string, template string, args ...interface{}) string {
	var template1 string
	if name == "" {
		template1 = level + " " + template
	} else {
		template1 = level + " [" + name + "] " + template
	}

	return fmt.Sprintf(template1, args...)
}

func makeLogEntry(ctx context.Context, level string, name string, template string, args ...interface{}) LogEntry {
	entry := LogEntry{
		Severity:  strings.TrimSpace(level),
		Timestamp: time.Now(),
		Message:   fmt.Sprintf(template, args...),
	}

	labels := map[string]string{
		"appname":    "trustedassertions",
		"loggername": name,
	}

	data, ok := appcontext.ContextData(ctx)
	if ok {
		labels["reqPath"] = data.ReqPath
		labels["traceparent"] = data.TraceParent
		entry.HttpRequest.RequestUrl = data.ReqPath
		entry.HttpRequest.RequestMethod = data.ReqMethod
		entry.TraceID, entry.SpanID = traceId(data.TraceParent)
		entry.Sampled = true
	}

	entry.Labels = labels
	return entry
}

func traceId(traceparent string) (string, string) {
	if traceparent == "" {
		return "", ""
	}
	parts := strings.Split(traceparent, "-")
	return "projects/trustedassertions/traces/" + parts[1], parts[2]
}

func Fatal(err error) {
	Errorf("Fatal error: %v", err)
	exitHandler(1)
}
