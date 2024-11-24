package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBasicLogging(t *testing.T) {
	var buf bytes.Buffer
	LogWriter = &buf
	StructureLogs = false

	Info("testing")

	if buf.String() != "INFO  testing\n" {
		t.Error("Did not find expected log entry")
	}
}

func TestErrorLogging(t *testing.T) {
	var buf bytes.Buffer
	LogWriter = &buf
	StructureLogs = false

	Errorf("Error %v", errors.New("Bang"))

	output := buf.String()
	if output != "ERROR Error Bang\n" {
		t.Errorf("Did not find expected log entry: %s", output)
	}
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

	log.Print("Logger Print")
	log.Debugf("Logger Debugf")
	log.Infof("Logger Infof")
	log.Errorf("Logger Errorf")

	if !strings.Contains(buf.String(), "\"message\":\"Logger Print\"") {
		t.Errorf("Did not find `Logger Print` in `%s`", buf.String())
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
