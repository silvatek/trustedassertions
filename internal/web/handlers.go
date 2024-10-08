package web

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

var errorMessages map[string]string

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/", HomeWebHandler)
	r.HandleFunc("/web/statements/{key}", ViewStatementWebHandler)
	r.HandleFunc("/web/entities/{key}", ViewEntityWebHandler)
	r.HandleFunc("/web/assertions/{key}", ViewAssertionWebHandler)
	r.HandleFunc("/web/broken", ErrorTestHandler)
	r.HandleFunc("/web/error", ErrorPageHandler)
	r.HandleFunc("/web/newstatement", NewStatementWebHandler)

	staticDir := http.Dir("./web/static")
	fs := http.FileServer(staticDir)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	errorMessages = make(map[string]string)
}

func RenderWebPage(pageName string, data interface{}, w http.ResponseWriter) {
	dir := "./web"

	t, err := template.ParseFiles(dir+"/"+"base.html", dir+"/"+pageName+".html")
	if err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("Error parsing template: %+v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("template.Execute: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func HomeWebHandler(w http.ResponseWriter, r *http.Request) {
	RenderWebPage("index", "", w)
}

func ViewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	statement, _ := datastore.ActiveDataStore.FetchStatement(assertions.MakeUri(key, "statement"))

	refs, _ := datastore.ActiveDataStore.FetchRefs(statement.Uri())

	data := struct {
		Uri        string
		Content    string
		ApiLink    string
		References []assertions.HashUri
	}{
		Uri:        statement.Uri().String(),
		Content:    statement.Content(),
		ApiLink:    statement.Uri().ApiPath(),
		References: refs,
	}

	RenderWebPage("viewstatement", data, w)
}

func ViewAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	uri := assertions.MakeUri(key, "assertion")
	assertion, _ := datastore.ActiveDataStore.FetchAssertion(uri)

	issuerUri := assertions.UriFromString(assertion.Issuer)
	if !issuerUri.HasType() {
		issuerUri = issuerUri.WithType("entity")
	}

	subjectUri := assertions.UriFromString(assertion.Subject)
	if !subjectUri.HasType() {
		subjectUri = subjectUri.WithType("statement")
	}

	data := struct {
		Uri         string
		Assertion   assertions.Assertion
		IssuerLink  string
		SubjectLink string
		ApiLink     string
		References  []string
	}{
		Uri:         assertion.Uri().String(),
		Assertion:   assertion,
		ApiLink:     assertion.Uri().ApiPath(),
		IssuerLink:  issuerUri.WebPath(),
		SubjectLink: subjectUri.WebPath(),
		References:  make([]string, 0),
	}

	RenderWebPage("viewassertion", data, w)
}

func ViewEntityWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]

	uri := assertions.MakeUri(key, "entity")
	entity, err := datastore.ActiveDataStore.FetchEntity(uri)
	if err != nil {
		HandleError(1001, "Unable to fetch entity", w, r)
		return
	}

	refs, _ := datastore.ActiveDataStore.FetchRefs(entity.Uri())

	data := struct {
		Uri        string
		CommonName string
		ApiLink    string
		PublicKey  string
		References []assertions.HashUri
	}{
		Uri:        uri.String(),
		CommonName: entity.CommonName,
		PublicKey:  fmt.Sprintf("%v", entity.PublicKey),
		ApiLink:    uri.ApiPath(),
		References: refs,
	}

	RenderWebPage("viewentity", data, w)
}

func fetchDefaultEntity() (assertions.Entity, error) {
	return datastore.ActiveDataStore.FetchEntity(assertions.UriFromString(os.Getenv("DEFAULT_ENTITY")))
}

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

func NewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		RenderWebPage("newstatementform", "", w)
	} else if r.Method == "POST" {
		log.Info("Creating new statement and assertion")
		r.ParseForm()
		content := r.Form.Get("statement")
		log.Debugf("Web post of new statement: %s", content)

		// Fetch the default entity
		entity, err := fetchDefaultEntity()
		if err != nil {
			HandleError(1002, "Error fetching default entity", w, r)
			return
		}

		b64key, err := datastore.ActiveDataStore.FetchKey(entity.Uri())
		if err != nil {
			HandleError(1003, "Error fetching default entity private key", w, r)
			return
		}
		privateKey := assertions.EncodePrivateKey(b64key)

		// Create and save the statement
		statement := assertions.NewStatement(content)
		datastore.ActiveDataStore.Store(&statement)

		su := statement.Uri()

		// Create and save an assertion by the default entity that the statement is probably true
		assertion := assertions.NewAssertion("IsTrue")
		assertion.Subject = su.String()
		assertion.Confidence = 0.8
		assertion.SetAssertingEntity(entity)
		assertion.MakeJwt(privateKey)
		datastore.ActiveDataStore.Store(&assertion)

		addAssertionReferences(assertion)

		// Redirect the user to the assertion
		http.Redirect(w, r, assertion.Uri().WebPath(), http.StatusSeeOther)

		log.Infof("Redirecting to %s", assertion.Uri().WebPath())
	}
}

func addAssertionReferences(a assertions.Assertion) {
	datastore.ActiveDataStore.StoreRef(a.Uri(), assertions.UriFromString(a.Subject), "Assertion.Subject:Statement")
	datastore.ActiveDataStore.StoreRef(a.Uri(), assertions.UriFromString(a.Issuer), "Assertion.Issuer:Entity")
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

	RenderWebPage("error", data, w)
}

func errorMessage(errorCode string) string {
	message, ok := errorMessages[errorCode]
	if !ok {
		return "Unknown error [" + errorCode + "]"
	}
	return message + " [" + errorCode + "]"
}

func ErrorTestHandler(w http.ResponseWriter, r *http.Request) {
	HandleError(9999, "Fake error for testing", w, r)
}
