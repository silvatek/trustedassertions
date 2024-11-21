package appcontext

import (
	"context"
	"net/http"
)

type CtxData struct {
	ReqPath     string
	ReqMethod   string
	TraceParent string
}

type key int

const ctxDataKey key = 0

func NewWebContext(req *http.Request) context.Context {
	return WebContext(req.Context(), req)
}

// Custom context for web requests
func WebContext(parent context.Context, req *http.Request) context.Context {
	var data CtxData

	data.ReqPath = req.URL.Path
	data.ReqMethod = req.Method
	data.TraceParent = req.Header.Get("traceparent")

	return context.WithValue(parent, ctxDataKey, data)
}

// Custom context for server startup
func InitContext() context.Context {
	var data CtxData

	data.ReqPath = "{INIT}"
	data.ReqMethod = "{INIT}"
	data.TraceParent = ""

	return context.WithValue(context.Background(), ctxDataKey, data)
}

// Returns the data associated with the context.
// If there is no associated data, returns an empty structure.
func ContextData(ctx context.Context) (CtxData, bool) {
	val := ctx.Value(ctxDataKey)
	if val == nil {
		return CtxData{}, false
	}

	return val.(CtxData), true
}
