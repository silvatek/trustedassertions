package web

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"text/template"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

var errorMessages map[string]string
var TemplateDir string
var DefaultEntityUri assertions.HashUri

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/", HomeWebHandler)
	r.HandleFunc("/web/statements/{key}", ViewStatementWebHandler)
	r.HandleFunc("/web/entities/{key}", ViewEntityWebHandler)
	r.HandleFunc("/web/assertions/{key}", ViewAssertionWebHandler)
	r.HandleFunc("/web/broken", ErrorTestHandler)
	r.HandleFunc("/web/error", ErrorPageHandler)
	r.HandleFunc("/web/newstatement", NewStatementWebHandler)

	addAuthHandlers(r)

	staticDir := http.Dir(TemplateDir + "/static")
	fs := http.FileServer(staticDir)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	errorMessages = make(map[string]string)
}

type PageData struct {
	AuthUser string
	LoggedIn bool
	Detail   interface{}
}

func RenderWebPage(pageName string, data interface{}, w http.ResponseWriter, r *http.Request) {
	dir := TemplateDir

	t, err := template.ParseFiles(dir+"/"+"base.html", dir+"/"+pageName+".html")
	if err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("Error parsing template: %+v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	username := authUser(r)
	pageData := PageData{
		AuthUser: username,
		LoggedIn: username != "",
		Detail:   data,
	}

	if err := t.ExecuteTemplate(w, "base", pageData); err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("template.Execute: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

// Returns the name of the currently authenticated user, or an empty string.
func authUser(r *http.Request) string {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return ""
	}
	userToken, err := jwt.Parse(cookie.Value,
		func(token *jwt.Token) (interface{}, error) {
			return userJwtKey, nil
		},
	)
	if err != nil {
		log.Errorf("Error parsing user JWT: %v", err)
		return ""
	}
	userName, _ := userToken.Claims.GetSubject()
	return userName
}

func HomeWebHandler(w http.ResponseWriter, r *http.Request) {
	RenderWebPage("index", "", w, r)
}

type ReferenceSummary struct {
	Uri     assertions.HashUri
	Summary string
}

func summariseAssertion(assertion assertions.Assertion, currentUri assertions.HashUri) string {
	if assertion.Issuer == "" {
		var err error
		assertion, err = assertions.ParseAssertionJwt(assertion.Content())
		if err != nil {
			return "Error parsing JWT"
		}
	}

	subjectUri := assertions.UriFromString(assertion.Subject)
	if subjectUri.Equals(currentUri) {
		issuer, _ := datastore.ActiveDataStore.FetchEntity(assertions.UriFromString(assertion.Issuer))
		return fmt.Sprintf("%s asserts this %s", issuer.CommonName, assertion.Category)
	}

	issuerUri := assertions.UriFromString(assertion.Issuer)
	if issuerUri.Equals(currentUri) {
		statement, _ := datastore.ActiveDataStore.FetchStatement(assertions.UriFromString(assertion.Subject))
		return fmt.Sprintf("This entity asserts that statement %s %s", statement.Uri().Short(), assertion.Category)
	}

	return "Some kind of assertion"
}

func enrichReferences(uris []assertions.HashUri, currentUri assertions.HashUri) []ReferenceSummary {
	summaries := make([]ReferenceSummary, 0)

	for _, uri := range uris {
		var summary string
		switch uri.Kind() {
		case "assertion":
			assertion, _ := datastore.ActiveDataStore.FetchAssertion(uri)

			summary = summariseAssertion(assertion, currentUri)
		default:
			summary = "unknown " + uri.Kind()
		}
		ref := ReferenceSummary{Uri: uri, Summary: summary}
		summaries = append(summaries, ref)
	}

	return summaries
}

func ViewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	statement, _ := datastore.ActiveDataStore.FetchStatement(assertions.MakeUri(key, "statement"))

	refUris, _ := datastore.ActiveDataStore.FetchRefs(statement.Uri())
	refs := enrichReferences(refUris, statement.Uri())

	data := struct {
		Uri        string
		ShortUri   string
		Content    string
		ApiLink    string
		References []ReferenceSummary
	}{
		Uri:        statement.Uri().String(),
		ShortUri:   statement.Uri().Short(),
		Content:    statement.Content(),
		ApiLink:    statement.Uri().ApiPath(),
		References: refs,
	}

	RenderWebPage("viewstatement", data, w, r)
}

func ViewAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	uri := assertions.MakeUri(key, "assertion")
	assertion, _ := datastore.ActiveDataStore.FetchAssertion(uri)

	issuerUri := assertions.UriFromString(assertion.Issuer)
	if !issuerUri.HasType() {
		issuerUri = issuerUri.WithType("entity")
	}

	issuer, _ := datastore.ActiveDataStore.FetchEntity(issuerUri)

	subjectUri := assertions.UriFromString(assertion.Subject)
	if !subjectUri.HasType() {
		subjectUri = subjectUri.WithType("statement")
	}

	subject, _ := datastore.ActiveDataStore.FetchStatement(subjectUri)

	data := struct {
		Uri         string
		ShortUri    string
		Assertion   assertions.Assertion
		IssuerLink  string
		IssuerName  string
		SubjectLink string
		SubjectText string
		ApiLink     string
		References  []string
	}{
		Uri:         assertion.Uri().String(),
		ShortUri:    assertion.Uri().Short(),
		Assertion:   assertion,
		ApiLink:     assertion.Uri().ApiPath(),
		IssuerLink:  issuerUri.WebPath(),
		IssuerName:  issuer.CommonName,
		SubjectLink: subjectUri.WebPath(),
		SubjectText: subject.Content(),
		References:  make([]string, 0),
	}

	RenderWebPage("viewassertion", data, w, r)
}

func ViewEntityWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]

	uri := assertions.MakeUri(key, "entity")
	entity, err := datastore.ActiveDataStore.FetchEntity(uri)
	if err != nil {
		HandleError(1001, "Unable to fetch entity", w, r)
		return
	}

	refUris, _ := datastore.ActiveDataStore.FetchRefs(entity.Uri())
	refs := enrichReferences(refUris, entity.Uri())

	data := struct {
		Uri        string
		ShortUri   string
		CommonName string
		ApiLink    string
		PublicKey  string
		References []ReferenceSummary
	}{
		Uri:        uri.String(),
		ShortUri:   uri.Short(),
		CommonName: entity.CommonName,
		PublicKey:  fmt.Sprintf("%v", entity.PublicKey),
		ApiLink:    uri.ApiPath(),
		References: refs,
	}

	RenderWebPage("viewentity", data, w, r)
}

func fetchDefaultEntity() (assertions.Entity, error) {
	return datastore.ActiveDataStore.FetchEntity(DefaultEntityUri)
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
		if authUser(r) == "" {
			HandleError(1005, "Not logged in", w, r)
			return
		}
		RenderWebPage("newstatementform", "", w, r)
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
		privateKey := assertions.StringToPrivateKey(b64key)

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
	HandleError(9999, "Fake error for testing", w, r)
}
