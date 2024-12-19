package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/appcontext"
)

func TestBasicLogging(t *testing.T) {
	var buf bytes.Buffer
	LogWriter = &buf
	StructureLogs = false

	Info("testing")

	assertStringsEqual(t, "Error in basic logging", buf.String(), "INFO  testing\n")
}

func TestErrorLogging(t *testing.T) {
	var buf bytes.Buffer
	LogWriter = &buf
	StructureLogs = false

	Errorf("Error %v", errors.New("Bang"))

	assertStringsEqual(t, "Error in logging arguments", buf.String(), "ERROR Error Bang\n")
}

func TestFatalLogging(t *testing.T) {
	var buf bytes.Buffer
	LogWriter = &buf
	StructureLogs = false

	exitCode := 99999

	exitHandler = func(code int) {
		exitCode = code
	}

	Fatal(errors.New("Bang"))

	output := buf.String()
	if output != "ERROR Fatal error: Bang\n" {
		t.Errorf("Did not find expected log entry: %s", output)
	}
	if exitCode != 1 {
		t.Errorf("Unexpected exit code %d", exitCode)
	}
}

func TestDebugLogging(t *testing.T) {
	var buf bytes.Buffer
	LogWriter = &buf
	StructureLogs = false

	Debug("Detail")

	output := buf.String()
	if output != "DEBUG Detail\n" {
		t.Errorf("Did not find expected log entry: %s", output)
	}
}

func TestStructuredLogs(t *testing.T) {
	original := StructureLogs

	var buf bytes.Buffer
	LogWriter = &buf
	StructureLogs = true

	Print("testing")

	values := make(map[string]interface{})
	json.Unmarshal(buf.Bytes(), &values)

	if values["message"] != "testing" {
		t.Errorf("Unexpected message: %v", values["message"])
	}

	StructureLogs = original
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer

	log := GetLogger("testing")
	log.Structured = true
	log.Writer = &buf

	log.Print("Logger", "Print")
	log.Println("Logger Println")
	log.Printf("Logger %s", "Printf")
	log.Debugf("Logger Debugf")
	log.Infof("Logger Infof")
	log.Errorf("Logger Errorf")

	if !strings.Contains(buf.String(), "\"message\":\"Logger Print\"") {
		t.Errorf("Did not find `Logger Print` in `%s`", buf.String())
	}
	if !strings.Contains(buf.String(), "\"message\":\"Logger Println\"") {
		t.Errorf("Did not find `Logger Println` in `%s`", buf.String())
	}
	if !strings.Contains(buf.String(), "\"message\":\"Logger Printf\"") {
		t.Errorf("Did not find `Logger Printf` in `%s`", buf.String())
	}
	if !strings.Contains(buf.String(), "\"message\":\"Logger Debugf\"") {
		t.Errorf("Did not find `Logger Debugf` in `%s`", buf.String())
	}
	if !strings.Contains(buf.String(), "\"message\":\"Logger Infof\"") {
		t.Errorf("Did not find `Logger Infof` in `%s`", buf.String())
	}
	if !strings.Contains(buf.String(), "\"message\":\"Logger Errorf\"") {
		t.Errorf("Did not find `Logger Errorf` in `%s`", buf.String())
	}
}

func TestSimpleLogging(t *testing.T) {
	assertStringsEqual(t, "Error in simple logging all", simpleLogMessage("logger", "INFO", "Colour: %s", "blue"), "INFO [logger] Colour: blue")
	assertStringsEqual(t, "Error in simple logging no name", simpleLogMessage("", "INFO", "Colour: %s", "blue"), "INFO Colour: blue")
	assertStringsEqual(t, "Error in simple logging no params", simpleLogMessage("logger", "INFO", "No params"), "INFO [logger] No params")
}

func TestParseTraceId(t *testing.T) {
	trace, span := traceId("")
	if trace != "" || span != "" {
		t.Errorf("Unexpected trace or span: %s / %s", trace, span)
	}

	trace, span = traceId("trace-test-123")
	if trace != "projects/trustedassertions/traces/test" || span != "123" {
		t.Errorf("Incorrect trace or span: %s / %s", trace, span)
	}
}

func TestLogEntryWithContext(t *testing.T) {
	ctx := appcontext.InitContext()

	entry := makeLogEntry(ctx, "INFO ", "", "Testing")

	assertStringsEqual(t, "Incorrect message in LogEntry", "Testing", fmt.Sprintf("%s", entry.Message))
	assertStringsEqual(t, "Incorrect severity in LogEntry", "INFO", entry.Severity)
	assertStringsEqual(t, "Incorrect trace ID in LogEntry", "", entry.TraceID)
}

func assertStringsEqual(t *testing.T, errorMessage string, actual string, expected string) {
	if actual != expected {
		t.Errorf(errorMessage+" `%s` != `%s`", actual, expected)
	}
}
