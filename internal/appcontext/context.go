package appcontext

import (
	"context"
	"net/http"
)

type CtxData struct {
	Dummy   string
	ReqPath string
}

type key int

const ctxDataKey key = 0

func NewWebContext(req *http.Request) context.Context {
	return WebContext(context.Background(), req)
}

func WebContext(parent context.Context, req *http.Request) context.Context {
	var data CtxData
	data.Dummy = "WebContext"
	data.ReqPath = req.URL.Path
	return context.WithValue(parent, ctxDataKey, data)
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
