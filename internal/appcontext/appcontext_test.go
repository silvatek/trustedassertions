package appcontext

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestDefaultContext(t *testing.T) {
	_, found := ContextData(context.TODO())

	if found {
		t.Error("Found context data in TODO context")
	}
}

func TestWebContext(t *testing.T) {
	req := http.Request{URL: &url.URL{Path: "/testing"}}

	ctx := NewWebContext(&req)

	data, found := ContextData(ctx)

	if !found {
		t.Error("Context data not found in WebContext")
	}
	if data.ReqPath != "/testing" {
		t.Errorf("Unexpected web context request path: %s", data.ReqPath)
	}
}

func TestInitContext(t *testing.T) {
	ctx := InitContext()
	data, found := ContextData(ctx)

	if !found {
		t.Error("Context data not found in InitContext")
	}
	if data.ReqPath != "{INIT}" {
		t.Errorf("Unexpected init context request path: %s", data.ReqPath)
	}
}
