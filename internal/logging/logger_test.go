package logging

import (
	"bytes"
	"encoding/json"
	"errors"
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
