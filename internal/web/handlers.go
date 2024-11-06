package web

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"text/template"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	log "silvatek.uk/trustedassertions/internal/logging"
)

var errorMessages map[string]string
var TemplateDir string
var DefaultEntityUri assertions.HashUri

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/", HomeWebHandler)
	r.HandleFunc("/web/statements/{hash}", ViewStatementWebHandler)
	r.HandleFunc("/web/entities/{hash}", ViewEntityWebHandler)
	r.HandleFunc("/web/assertions/{hash}", ViewAssertionWebHandler)
	r.HandleFunc("/web/documents/{hash}", ViewDocumentWebHandler)
	r.HandleFunc("/web/broken", ErrorTestHandler)
	r.HandleFunc("/web/error", ErrorPageHandler)
	r.HandleFunc("/web/newstatement", NewStatementWebHandler)
	r.HandleFunc("/web/newentity", NewEntityWebHandler)
	r.HandleFunc("/web/statements/{hash}/addassertion", AddStatementAssertionWebHandler)
	r.HandleFunc("/web/search", SearchWebHandler)
	r.HandleFunc("/web/share", SharePageWebHandler)
	r.HandleFunc("/web/qrcode", qrCodeGenerator)

	r.NotFoundHandler = http.HandlerFunc(NotFoundWebHandler)

	addAuthHandlers(r)

	staticDir := http.Dir(TemplateDir + "/static")
	fs := http.FileServer(staticDir)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	errorMessages = make(map[string]string)
}

type PageData struct {
	LeftMenu  PageMenu
	RightMenu PageMenu
	AuthUser  string
	AuthName  string
	LoggedIn  bool
	Detail    interface{}
}

func RenderWebPageWithStatus(ctx context.Context, pageName string, data interface{}, menu []PageMenuItem, w http.ResponseWriter, r *http.Request, status int) {
	log.DebugfX(ctx, "Rendering page %s", pageName)
	dir := TemplateDir

	t, err := template.ParseFiles(dir+"/"+"base.html", dir+"/"+pageName+".html")
	if err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("Error parsing template: %+v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	log.DebugfX(ctx, "Templates parsed")

	username := authUser(r)
	pageData := PageData{
		AuthUser: username,
		AuthName: nameOnly(username),
		LoggedIn: username != "",
		Detail:   data,
	}

	// if username != "" && pageName != "loggedout" {
	SetAuthCookie(username, w) // Refresh the auth cookie
	// }

	leftMenu := PageMenu{}
	if pageName != "index" {
		leftMenu.AddLink("Home", "/")
	}
	for _, item := range menu {
		leftMenu.AddItem(&item)
	}

	pageData.LeftMenu = leftMenu

	rightMenu := PageMenu{}

	if pageName == "loggedout" || pageName == "loginform" {
		//
	} else if username == "" {
		rightMenu.AddRightLink("Login", "/web/login")
	} else {
		rightMenu.AddRightText(nameOnly(username))
		rightMenu.AddRightLink("Logout", "/web/logout")
	}
	pageData.RightMenu = rightMenu

	if status != 0 {
		w.WriteHeader(status)
	}

	if err := t.ExecuteTemplate(w, "base", pageData); err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.Errorf("template.Execute: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}

	log.DebugfX(ctx, "Page rendered")
}

func RenderWebPage(ctx context.Context, pageName string, data interface{}, menu []PageMenuItem, w http.ResponseWriter, r *http.Request) {
	RenderWebPageWithStatus(ctx, pageName, data, menu, w, r, 200)
}

func nameOnly(username string) string {
	n := strings.Index(username, "@")
	if n == -1 {
		return username
	} else {
		return username[0:n]
	}
}

// Returns the name of the currently authenticated user, or an empty string.
func authUser(r *http.Request) string {
	cookie, err := r.Cookie("auth")
	if err != nil {
		return ""
	}
	if cookie.Value == "" {
		return ""
	}
	userName, err := auth.ParseUserJwt(cookie.Value, userJwtKey)
	if err != nil {
		log.Errorf("Error parsing user JWT: %v", err)
		return ""
	}
	return userName
}

func HomeWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	log.DebugfX(ctx, "Home page accessed")

	data := struct {
		DefaultDocument assertions.HashUri
	}{
		DefaultDocument: assertions.DefaultDocumentUri,
	}

	RenderWebPage(ctx, "index", data, nil, w, r)
}

func ViewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	key := mux.Vars(r)["hash"]
	statement, _ := datastore.ActiveDataStore.FetchStatement(ctx, assertions.MakeUri(key, "statement"))

	refUris, _ := datastore.ActiveDataStore.FetchRefs(ctx, statement.Uri())
	refs := assertions.EnrichReferences(ctx, refUris, statement.Uri(), datastore.ActiveDataStore)

	data := struct {
		Uri        assertions.HashUri
		ShortUri   string
		Content    string
		ApiLink    string
		References []assertions.ReferenceSummary
	}{
		Uri:        statement.Uri(),
		ShortUri:   statement.Uri().Short(),
		Content:    statement.Content(),
		ApiLink:    statement.Uri().ApiPath(),
		References: refs,
	}

	menu := []PageMenuItem{
		{Text: "Raw", Target: statement.Uri().ApiPath()},
		{Text: "Share", Target: "/web/share?hash=" + statement.Uri().Hash() + "&type=statement"},
	}

	RenderWebPage(ctx, "viewstatement", data, menu, w, r)
}

func ViewAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	key := mux.Vars(r)["hash"]
	uri := assertions.MakeUri(key, "assertion")
	assertion, _ := datastore.ActiveDataStore.FetchAssertion(ctx, uri)

	issuerUri := assertions.UriFromString(assertion.Issuer)
	if !issuerUri.HasType() {
		issuerUri = issuerUri.WithType("entity")
	}

	issuer, _ := datastore.ActiveDataStore.FetchEntity(ctx, issuerUri)

	subjectUri := assertions.UriFromString(assertion.Subject)
	if !subjectUri.HasType() {
		subjectUri = subjectUri.WithType("statement")
	}

	subject, _ := datastore.ActiveDataStore.FetchStatement(ctx, subjectUri)

	refUris, _ := datastore.ActiveDataStore.FetchRefs(ctx, uri)
	refs := assertions.EnrichReferences(ctx, refUris, uri, datastore.ActiveDataStore)

	data := struct {
		Uri         string
		ShortUri    string
		Assertion   assertions.Assertion
		IssuerLink  string
		IssuerName  string
		SubjectLink string
		SubjectText string
		ApiLink     string
		References  []assertions.ReferenceSummary
	}{
		Uri:         assertion.Uri().String(),
		ShortUri:    assertion.Uri().Short(),
		Assertion:   assertion,
		ApiLink:     assertion.Uri().ApiPath(),
		IssuerLink:  issuerUri.WebPath(),
		IssuerName:  issuer.CommonName,
		SubjectLink: subjectUri.WebPath(),
		SubjectText: subject.Content(),
		References:  refs,
	}

	menu := []PageMenuItem{
		{Text: "Raw", Target: assertion.Uri().ApiPath()},
		{Text: "Share", Target: "/web/share?hash=" + assertion.Uri().Hash() + "&type=assertion"},
	}

	RenderWebPage(ctx, "viewassertion", data, menu, w, r)
}

func ViewEntityWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	key := mux.Vars(r)["hash"]

	uri := assertions.MakeUri(key, "entity")
	entity, err := datastore.ActiveDataStore.FetchEntity(ctx, uri)
	if err != nil {
		HandleError(ErrorEntityFetch, "Unable to fetch entity", w, r)
		return
	}

	refUris, _ := datastore.ActiveDataStore.FetchRefs(ctx, entity.Uri())
	refs := assertions.EnrichReferences(ctx, refUris, entity.Uri(), datastore.ActiveDataStore)

	data := struct {
		Uri        string
		ShortUri   string
		CommonName string
		ApiLink    string
		PublicKey  string
		References []assertions.ReferenceSummary
	}{
		Uri:        uri.String(),
		ShortUri:   uri.Short(),
		CommonName: entity.CommonName,
		PublicKey:  fmt.Sprintf("%v", entity.PublicKey),
		ApiLink:    uri.ApiPath(),
		References: refs,
	}

	menu := []PageMenuItem{
		{Text: "Raw", Target: entity.Uri().ApiPath()},
		{Text: "Share", Target: "/web/share?hash=" + entity.Uri().Hash() + "&type=entity"},
	}

	RenderWebPage(ctx, "viewentity", data, menu, w, r)
}

func ViewDocumentWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["hash"]
	document, _ := datastore.ActiveDataStore.FetchDocument(ctx, assertions.MakeUri(key, "document"))

	data := struct {
		Doc       assertions.Document
		Title     string
		DocHtml   string
		AuthorUri assertions.HashUri
	}{
		Doc:       document,
		Title:     "Testing",
		DocHtml:   document.ToHtml(),
		AuthorUri: assertions.UriFromString(document.Metadata.Author.Entity),
	}

	// menu := []PageMenuItem{
	// 	{Text: "Raw", Target: document.Uri().ApiPath()},
	// 	{Text: "Share", Target: "/web/share?hash=" + document.Uri().Hash() + "&type=statement"},
	// }

	RenderWebPage(ctx, "viewdocument", data, nil, w, r)
}

func NewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	username := authUser(r)
	if username == "" {
		HandleError(ErrorNoAuth, "Not logged in", w, r)
		return
	}
	user, err := datastore.ActiveDataStore.FetchUser(username)
	if err != nil {
		HandleError(ErrorUserNotFound, "User not found", w, r)
		return
	}

	if r.Method == "GET" {
		RenderWebPage(ctx, "newstatementform", user, nil, w, r)
	} else if r.Method == "POST" {
		log.InfofX(ctx, "Creating new statement and assertion")
		r.ParseForm()
		content := r.Form.Get("statement")
		log.DebugfX(ctx, "Web post of new statement: %s", content)

		keyId := r.Form.Get("sign_as")
		log.DebugfX(ctx, "Signing key ID: %s", keyId)

		confidence, _ := strconv.ParseFloat(r.Form.Get("confidence"), 32)

		keyUri := assertions.UriFromString(keyId)

		if !user.HasKey(keyId) {
			HandleError(ErrorKeyAccess, "User does not have access to selected signing key", w, r)
			return
		}

		assertion, err := datastore.CreateStatementAndAssertion(ctx, content, keyUri, confidence)
		if err != nil {
			HandleError(ErrorMakeAssertion, "Error making new statement and assertion", w, r)
			return
		}

		// Redirect the user to the assertion
		http.Redirect(w, r, assertion.Uri().WebPath(), http.StatusSeeOther)

		log.Infof("Redirecting to %s", assertion.Uri().WebPath())
	}
}

func SearchWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	query := r.URL.Query().Get("query")

	query, _ = url.QueryUnescape(query)

	results, _ := datastore.ActiveDataStore.Search(query)

	data := struct {
		Query   string
		Results []datastore.SearchResult
	}{
		Query:   query,
		Results: results,
	}

	RenderWebPage(ctx, "searchresults", data, nil, w, r)
}

func NewEntityWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	username := authUser(r)
	if username == "" {
		HandleError(ErrorNoAuth, "Not logged in", w, r)
		return
	}
	user, err := datastore.ActiveDataStore.FetchUser(username)
	if err != nil {
		HandleError(ErrorUserNotFound, "User not found", w, r)
		return
	}

	if r.Method == "GET" {
		RenderWebPage(ctx, "newentityform", user, nil, w, r)
	} else if r.Method == "POST" {
		log.InfofX(ctx, "Creating new entity and signing key")
		r.ParseForm()
		commonName := r.Form.Get("commonname")
		log.DebugfX(ctx, "Common name: %s", commonName)

		privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
		entity := assertions.Entity{CommonName: commonName}
		entity.MakeCertificate(privateKey)

		datastore.ActiveDataStore.Store(ctx, &entity)

		datastore.ActiveDataStore.StoreKey(entity.Uri(), assertions.PrivateKeyToString(privateKey))

		user.AddKeyRef(entity.Uri().Escaped(), entity.CommonName)
		datastore.ActiveDataStore.StoreUser(user)

		// Redirect the user to the assertion
		http.Redirect(w, r, entity.Uri().WebPath(), http.StatusSeeOther)

		log.Infof("Redirecting to %s", entity.Uri().WebPath())
	}
}

func AddStatementAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	username := authUser(r)
	if username == "" {
		HandleError(ErrorNoAuth, "Not logged in", w, r)
		return
	}
	user, err := datastore.ActiveDataStore.FetchUser(username)
	if err != nil {
		HandleError(ErrorUserNotFound, "User not found", w, r)
		return
	}

	statementHash := mux.Vars(r)["hash"]

	if r.Method == "GET" {
		statement, err := datastore.ActiveDataStore.FetchStatement(ctx, assertions.MakeUri(statementHash, "statement"))
		if err != nil {
			log.Errorf("Error fetching statement: %v", err)
		} else {
			log.Debugf("Statement content = %s", statement.Content())
		}

		data := struct {
			Statement assertions.Statement
			User      auth.User
		}{
			Statement: statement,
			User:      user,
		}

		RenderWebPage(ctx, "addassertionform", data, nil, w, r)
	} else if r.Method == "POST" {
		log.InfofX(ctx, "Creating new assertion for statement")
		r.ParseForm()

		keyId := r.Form.Get("sign_as")
		log.DebugfX(ctx, "Signing key ID: %s", keyId)

		keyUri := assertions.UriFromString(keyId)

		if !user.HasKey(keyId) {
			HandleError(ErrorKeyAccess, "User does not have access to selected signing key", w, r)
			return
		}

		b64key, err := datastore.ActiveDataStore.FetchKey(keyUri)
		if err != nil {
			HandleError(ErrorKeyFetch, "Error fetching entity private key", w, r)
			return
		}
		privateKey := assertions.StringToPrivateKey(b64key)

		entity, _ := datastore.ActiveDataStore.FetchEntity(ctx, keyUri)

		su := assertions.MakeUri(statementHash, "statement")

		confidence, _ := strconv.ParseFloat(r.Form.Get("confidence"), 32)
		kind := r.Form.Get("assertion_type")

		assertion := datastore.CreateAssertion(ctx, su, confidence, entity, privateKey, kind)

		// Redirect the user to the assertion
		http.Redirect(w, r, assertion.Uri().WebPath(), http.StatusSeeOther)

		log.DebugfX(ctx, "Redirecting to %s", assertion.Uri().WebPath())
	}
}
