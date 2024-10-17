package web

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"

	log "silvatek.uk/trustedassertions/internal/logging"
)

const ErrorEntityFetch = 1001

const ErrorNoAuth = 2001
const ErrorUserNotFound = 2002
const ErrorKeyFetch = 2003
const ErrorKeyAccess = 2004
const ErrorAuthFail = 2005

const ErrorFakeTest = 9999

// Error handling for web app.
//
// Logs an error with a message, code and unique ID, then redirects to the error page with the error code and ID.
func HandleError(errorCode int, errorMessage string, w http.ResponseWriter, r *http.Request) {
	errorInt, _ := rand.Int(rand.Reader, big.NewInt(0xFFFFFF))
	errorId := fmt.Sprintf("%X", errorInt)
	log.Errorf("%s [%d,%s]", errorMessage, errorCode, errorId)
	errorMessages[fmt.Sprintf("%d", errorCode)] = errorMessage
	errorPage := fmt.Sprintf("/web/error?err=%d&id=%s", errorCode, errorId)
	http.Redirect(w, r, errorPage, http.StatusSeeOther)
}

func ErrorPageHandler(w http.ResponseWriter, r *http.Request) {
	errorCode := r.URL.Query().Get("err")
	errorId := r.URL.Query().Get("id")

	data := struct {
		ErrorMessage string
		ErrorID      string
	}{
		ErrorMessage: errorMessage(errorCode),
		ErrorID:      errorId,
	}

	RenderWebPage("error", data, w, r)
}

func errorMessage(errorCode string) string {
	message, ok := errorMessages[errorCode]
	if !ok {
		return "Unknown error [" + errorCode + "]"
	}
	return message + " [" + errorCode + "]"
}

func ErrorTestHandler(w http.ResponseWriter, r *http.Request) {
	HandleError(ErrorFakeTest, "Fake error for testing", w, r)
}
