package testcontext

import "testing"

// TestContext is compatible with testing.T but can also be mocked.
type TestContext interface {
	Error(args ...any)
	Errorf(format string, args ...any)
}

type MockTestContext struct {
	ErrorsFound bool
}

func (t *MockTestContext) Error(args ...any) {
	t.ErrorsFound = true
}

func (t *MockTestContext) Errorf(format string, args ...any) {
	t.ErrorsFound = true
}

func (t *MockTestContext) AssertErrorsFound(t1 *testing.T) {
	if !t.ErrorsFound {
		t.Error("No errors reported to MockTestContext")
	}
}
