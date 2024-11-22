package web

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"

	"silvatek.uk/trustedassertions/internal/appcontext"
)

const FetchError = 1000
const ErrorMakeAssertion = 2002
const ErrorKeyFetch = 2003
const ErrorKeyAccess = 2004

const AuthError = 3000
const ErrorUserNotFound = 3002
const ErrorAuthFail = 3005
const ErrorRegCode = 3101

type HandlerError struct {
	ErrorCode   int
	HttpCode    int
	UserMessage string
	LogMessage  string
	ErrorId     string
}

var ErrorEntityFetch = HandlerError{ErrorCode: FetchError + 1, UserMessage: "Error retrieving entity"}
var ErrorAssertionFetch = HandlerError{ErrorCode: FetchError + 2, UserMessage: "Error retrieving assertion"}

var ErrorNoAuth = HandlerError{ErrorCode: AuthError + 1, UserMessage: "Not logged in"}

var ErrorFakeTest = HandlerError{ErrorCode: 9999, UserMessage: "Fake error for testing"}

var errorMessages map[string]string

func (e HandlerError) instance(logMessage string) HandlerError {
	if logMessage == "" {
		logMessage = e.UserMessage
	}
	if e.HttpCode == 0 {
		e.HttpCode = http.StatusInternalServerError
	}

	instance := HandlerError{
		ErrorCode:   e.ErrorCode,
		HttpCode:    e.HttpCode,
		UserMessage: e.UserMessage,
		LogMessage:  logMessage,
		ErrorId:     makeErrorId(),
	}

	return instance
}

func HandleWebError(ctx context.Context, err HandlerError, w http.ResponseWriter, r *http.Request) {
	if err.ErrorId == "" {
		err = err.instance("")
	}
	log.ErrorfX(ctx, fmt.Sprintf("%d : %s (%s)", err.ErrorCode, err.LogMessage, err.ErrorId))
	errorMessages[fmt.Sprintf("%d", err.ErrorCode)] = err.UserMessage
	errorPage := fmt.Sprintf("/web/error?err=%d&id=%s", err.ErrorCode, err.ErrorId)
	http.Redirect(w, r, errorPage, http.StatusSeeOther)
}

// Error handling for web app.
//
// Logs an error with a message, code and unique ID, then redirects to the error page with the error code and ID.
func HandleError(errorCode int, errorMessage string, w http.ResponseWriter, r *http.Request) {
	errorId := makeErrorId()
	log.Errorf("%s [%d,%s]", errorMessage, errorCode, errorId)
	errorMessages[fmt.Sprintf("%d", errorCode)] = errorMessage
	errorPage := fmt.Sprintf("/web/error?err=%d&id=%s", errorCode, errorId)
	http.Redirect(w, r, errorPage, http.StatusSeeOther)
}

func makeErrorId() string {
	errorInt, _ := rand.Int(rand.Reader, big.NewInt(0xFFFFFF))
	return fmt.Sprintf("%X", errorInt)
}

func ErrorPageHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	errorCode := r.URL.Query().Get("err")
	errorId := r.URL.Query().Get("id")

	data := struct {
		ErrorMessage string
		ErrorID      string
	}{
		ErrorMessage: errorMessage(errorCode),
		ErrorID:      errorId,
	}

	RenderWebPageWithStatus(ctx, "error", data, nil, w, r, 500)
}

func errorMessage(errorCode string) string {
	message, ok := errorMessages[errorCode]
	if !ok {
		return "Unknown error [" + errorCode + "]"
	}
	return message + " [" + errorCode + "]"
}

func ErrorTestHandler(w http.ResponseWriter, r *http.Request) {
	HandleWebError(appcontext.NewWebContext(r), ErrorFakeTest.instance(""), w, r)
}

func NotFoundWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	RenderWebPageWithStatus(ctx, "notfound", "", nil, w, r, 404)
}
