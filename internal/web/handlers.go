package web

import (
	"net/http"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/", HomeWebHandler)
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

func ViewAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	assertion, _ := datastore.ActiveDataStore.FetchAssertion("hash://sha256/" + key)

	data := struct {
		Uri         string
		Assertion   assertions.Assertion
		IssuerLink  string
		SubjectLink string
	}{
		Uri:        assertion.Uri(),
		Assertion:  assertion,
		IssuerLink: "/api/v1/entities/" + strings.TrimPrefix(assertion.Issuer, "hash://sha256/"),
		//TODO: don't assume subject is a statement
		SubjectLink: "/api/v1/statements/" + strings.TrimPrefix(assertion.Subject, "hash://sha256/"),
	}

	RenderWebPage("viewassertion", data, w)
}

func NewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		RenderWebPage("newstatementform", "", w)
	} else if r.Method == "POST" {
		r.ParseForm()
		content := r.Form.Get("statement")
		log.Debugf("Web post of new statement: %s", content)

		// Create and save the statement
		statement := assertions.NewStatement(content)
		datastore.ActiveDataStore.Store(&statement)

		// Fetch the default entity
		entity, _ := datastore.ActiveDataStore.FetchEntity("hash://sha256/c6355ef5dfbc9da513ac4d683729ee24209aa1f9a8afe66bb4aa60217439183f")

		// Create and save an assertion by the default entity that the statement is probably true
		assertion := assertions.NewAssertion("IsTrue")
		assertion.Subject = statement.Uri()
		assertion.Confidence = 0.8
		assertion.SetAssertingEntity(entity)
		datastore.ActiveDataStore.Store(&assertion)

		// Redirect the user to the assertion
		hash := strings.TrimPrefix(assertion.Uri(), "hash://sha256/")
		http.Redirect(w, r, "/api/v1/assertions/"+hash, http.StatusSeeOther)
	}
}
