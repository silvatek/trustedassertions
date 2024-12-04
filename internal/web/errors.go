package web

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"

	"silvatek.uk/trustedassertions/internal/appcontext"
)

// Application errors, including external and internal messages, error codes and http status codes.
type AppError struct {
	ErrorCode   int       // A numeric code shared by all instances of this error
	HttpCode    int       // The Http status code that is appropriate for this error
	UserMessage string    // An error message suitable to be displayed to the end user
	LogMessage  string    // An error message intended to be written to server logs for debugging
	ErrorId     string    // A unique code for a specific instance of an error at a particular time for a particular user
	template    *AppError // The error on which an instance is based on
}

const FetchError = 1000

var ErrorEntityFetch = AppError{ErrorCode: FetchError + 1, UserMessage: "Error retrieving entity"}
var ErrorAssertionFetch = AppError{ErrorCode: FetchError + 2, UserMessage: "Error retrieving assertion"}

const UpdateError = 2000

var ErrorMakeAssertion = AppError{ErrorCode: UpdateError + 2, UserMessage: "Error making assertion"}
var ErrorKeyFetch = AppError{ErrorCode: UpdateError + 3, UserMessage: "Error fetching key"}
var ErrorKeyAccess = AppError{ErrorCode: UpdateError + 4, UserMessage: "Error accessing key", HttpCode: 403}
var ErrorParseDocument = AppError{ErrorCode: UpdateError + 5, UserMessage: "Error parsing document XML"}

var ErrorFakeTest = AppError{ErrorCode: 9999, UserMessage: "Fake error for testing"}

var errors map[string]AppError

// Create an instance of the error
func (e AppError) instance(logMessage string) AppError {
	if logMessage == "" {
		logMessage = e.UserMessage
	}
	if e.HttpCode == 0 {
		e.HttpCode = http.StatusInternalServerError
	}

	instance := AppError{
		ErrorCode:   e.ErrorCode,
		HttpCode:    e.HttpCode,
		UserMessage: e.UserMessage,
		LogMessage:  logMessage,
		ErrorId:     makeErrorId(),
		template:    &e,
	}

	return instance
}

func (e AppError) Error() string {
	if e.LogMessage == "" {
		return e.UserMessage
	} else {
		return e.LogMessage
	}
}

func (e AppError) Template() AppError {
	if e.template != nil {
		return *e.template
	} else {
		return e
	}
}

// Error handling for web app.
//
// Logs an error with a message, code and unique ID, then redirects to the error page with the error code and ID.
func HandleError(ctx context.Context, err AppError, w http.ResponseWriter, r *http.Request) {
	if err.ErrorId == "" {
		err = err.instance("")
	}
	log.ErrorfX(ctx, fmt.Sprintf("%d : %s (%s)", err.ErrorCode, err.Error(), err.ErrorId))
	errors[fmt.Sprintf("%d", err.ErrorCode)] = err.Template()
	errorPage := fmt.Sprintf("/web/error?err=%d&id=%s", err.ErrorCode, err.ErrorId)
	http.Redirect(w, r, errorPage, http.StatusSeeOther)
}

func makeErrorId() string {
	errorInt, _ := rand.Int(rand.Reader, big.NewInt(0xFFFFFF))
	return fmt.Sprintf("%X", errorInt)
}

// Handler for showing information about an error to the user.
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
	error, ok := errors[errorCode]
	if !ok {
		return "Unknown error [" + errorCode + "]"
	}
	return error.UserMessage + " [" + errorCode + "]"
}

func ErrorTestHandler(w http.ResponseWriter, r *http.Request) {
	HandleError(appcontext.NewWebContext(r), ErrorFakeTest.instance(""), w, r)
}

func NotFoundWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	RenderWebPageWithStatus(ctx, "notfound", "", nil, w, r, 404)
}
