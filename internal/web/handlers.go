package web

import (
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/", HomeWebHandler)
	r.HandleFunc("/web/statements/{key}", ViewStatementWebHandler)
	r.HandleFunc("/web/entities/{key}", ViewEntityWebHandler)
	r.HandleFunc("/web/assertions/{key}", ViewAssertionWebHandler)
	r.HandleFunc("/web/newstatement", NewStatementWebHandler)

	staticDir := http.Dir("./web/static")
	fs := http.FileServer(staticDir)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
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
		ApiLink:    "/api/v1/statements/" + key,
		References: refs,
	}

	RenderWebPage("viewstatement", data, w)
}

func ViewAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	assertion, _ := datastore.ActiveDataStore.FetchAssertion(assertions.MakeUri(key, "assertion"))

	data := struct {
		Uri         string
		Assertion   assertions.Assertion
		IssuerLink  string
		SubjectLink string
		ApiLink     string
		References  []string
	}{
		Uri:        assertion.Uri().String(),
		Assertion:  assertion,
		ApiLink:    "/api/v1/statements/" + key,
		IssuerLink: "/web/entities/" + assertions.UriHash(assertion.Issuer),
		//TODO: don't assume subject is a statement
		SubjectLink: "/web/statements/" + assertions.UriHash(assertion.Subject),
		References:  make([]string, 0),
	}

	RenderWebPage("viewassertion", data, w)
}

func ViewEntityWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]

	entity, err := datastore.ActiveDataStore.FetchEntity(assertions.MakeUri(key, "entity"))
	if err != nil {
		http.Redirect(w, r, "/error?err=1001", http.StatusSeeOther)
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
		Uri:        assertions.HashToUri(key, ""),
		CommonName: entity.CommonName,
		PublicKey:  fmt.Sprintf("%v", entity.PublicKey),
		ApiLink:    "/api/v1/entities/" + key,
		References: refs,
	}

	RenderWebPage("viewentity", data, w)
}

func fetchDefaultEntity() (assertions.Entity, error) {
	return datastore.ActiveDataStore.FetchEntity(assertions.MakeUri(os.Getenv("DEFAULT_ENTITY"), "entity"))
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
			http.Redirect(w, r, "/error?err=1002", http.StatusSeeOther)
			return
		}

		b64key, err := datastore.ActiveDataStore.FetchKey(entity.Uri())
		if err != nil {
			http.Redirect(w, r, "/error?err=1003", http.StatusSeeOther)
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
