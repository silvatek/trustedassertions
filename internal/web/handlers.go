package web

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"text/template"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/docs"
	"silvatek.uk/trustedassertions/internal/entities"
	"silvatek.uk/trustedassertions/internal/logging"
	ref "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/statements"
)

var TemplateDir string
var DefaultEntityUri ref.HashUri

var log = logging.GetLogger("web")

const HomePath = "/web/home"

func AddHandlers(r *mux.Router) {
	r.HandleFunc("/", HomeRedirectWebHandler)
	r.HandleFunc("/web", HomeRedirectWebHandler)
	r.HandleFunc("/web/home", HomeWebHandler)
	r.HandleFunc("/web/statements/{hash}", ViewStatementWebHandler)
	r.HandleFunc("/web/entities/{hash}", ViewEntityWebHandler)
	r.HandleFunc("/web/assertions/{hash}", ViewAssertionWebHandler)
	r.HandleFunc("/web/documents/{hash}", ViewDocumentWebHandler)
	r.HandleFunc("/web/broken", ErrorTestHandler)
	r.HandleFunc("/web/error", ErrorPageHandler)
	r.HandleFunc("/web/newstatement", NewStatementWebHandler)
	r.HandleFunc("/web/newentity", NewEntityWebHandler)
	r.HandleFunc("/web/newdocument", NewDocumentWebHandler)
	r.HandleFunc("/web/statements/{hash}/addassertion", AddStatementAssertionWebHandler)
	r.HandleFunc("/web/search", SearchWebHandler)
	r.HandleFunc("/web/share", SharePageWebHandler)
	r.HandleFunc("/web/qrcode", qrCodeGenerator)

	r.NotFoundHandler = http.HandlerFunc(NotFoundWebHandler)

	addAuthHandlers(r)
	AddAttackHandlers(r)

	r.PathPrefix("/web/static").Handler(StaticHandler())

	errors = make(map[string]AppError)
}

func StaticHandler() http.Handler {
	staticDir := http.Dir(TemplateDir + "/static")
	fs := http.FileServer(staticDir)
	h := http.StripPrefix("/web/static", CacheControlWrapper(fs))
	return h
}

func CacheControlWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetCacheControl(w, 5*60)
		h.ServeHTTP(w, r)
	})
}

type PageData struct {
	LeftMenu  PageMenu
	RightMenu PageMenu
	AuthUser  string
	AuthName  string
	LoggedIn  bool
	CsrfField interface{}
	Detail    interface{}
}

func RenderWebPageWithStatus(ctx context.Context, pageName string, data interface{}, menu []PageMenuItem, w http.ResponseWriter, r *http.Request, status int) {
	dir := TemplateDir

	t, err := template.ParseFiles(dir+"/"+"base.html", dir+"/"+pageName+".html")
	if err != nil {
		msg := http.StatusText(http.StatusInternalServerError)
		log.ErrorfX(ctx, "Error parsing template: %+v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	username := authUsername(r)
	pageData := PageData{
		AuthUser:  username,
		AuthName:  nameOnly(username),
		LoggedIn:  username != "",
		CsrfField: csrf.TemplateField(r),
		Detail:    data,
	}

	if r.URL.Path == HomePath {
		SetCacheControl(w, 10*60)
	} else {
		SetCacheControl(w, 0)
	}

	if pageName == "loggedout" {
		SetAuthCookie("", w)
	} else {
		SetAuthCookie(username, w) // Refresh the auth cookie
	}

	leftMenu := PageMenu{}
	if pageName != "index" {
		leftMenu.AddLink("Home", HomePath)
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
		rightMenu.AddRightLink("Register", "/web/register")
	} else {
		rightMenu.AddRightLink(nameOnly(username), "/web/profile")
		rightMenu.AddRightLink("Logout", "/web/logout")
	}
	pageData.RightMenu = rightMenu

	if status != 0 {
		w.WriteHeader(status)
	}

	if err := t.ExecuteTemplate(w, "base", pageData); err != nil {
		log.ErrorfX(ctx, "template.Execute: %v", err)
		msg := http.StatusText(http.StatusInternalServerError)
		http.Error(w, msg, http.StatusInternalServerError)
	}
}

func SetCacheControl(w http.ResponseWriter, cacheLifeInSeconds int) {
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(cacheLifeInSeconds))
}

func RenderWebPage(ctx context.Context, pageName string, data interface{}, menu []PageMenuItem, w http.ResponseWriter, r *http.Request) {
	RenderWebPageWithStatus(ctx, pageName, data, menu, w, r, 200)
}

func HomeRedirectWebHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, HomePath, http.StatusSeeOther)
}

func HomeWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	log.DebugfX(ctx, "Home page accessed")

	data := struct {
		DefaultDocument ref.HashUri
	}{
		DefaultDocument: docs.DefaultDocumentUri,
	}

	RenderWebPage(ctx, "index", data, nil, w, r)
}

func ViewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	key := mux.Vars(r)["hash"]
	statement, _ := datastore.ActiveDataStore.FetchStatement(ctx, ref.MakeUri(key, "statement"))

	refs, _ := datastore.ActiveDataStore.FetchRefs(ctx, statement.Uri())
	enrichReferencesTo(ctx, &statement, refs)

	data := struct {
		Uri        ref.HashUri
		ShortUri   string
		Content    string
		ApiLink    string
		References []ref.Reference
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

func enrichReferencesTo(ctx context.Context, target ref.Referenceable, refs []ref.Reference) {
	var wg sync.WaitGroup

	for n, reference := range refs {
		if reference.Summary == "" {
			wg.Add(1)
			go func(ref *ref.Reference) {
				// Construct a summary for the reference
				datastore.MakeReferenceSummary(ctx, &target, ref, datastore.ActiveDataStore)

				// Store the newly summarised reference back in the datastore
				datastore.ActiveDataStore.StoreRef(ctx, *ref)

				refs[n] = *ref
				wg.Done()
			}(&reference)
		}
	}

	wg.Wait()
}

func ViewAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	key := mux.Vars(r)["hash"]
	uri := ref.MakeUri(key, "assertion")
	assertion, _ := datastore.ActiveDataStore.FetchAssertion(ctx, uri)

	issuerUri := ref.UriFromString(assertion.Issuer)
	if !issuerUri.HasType() {
		issuerUri = issuerUri.WithType("entity")
	}

	issuer, _ := datastore.ActiveDataStore.FetchEntity(ctx, issuerUri)

	subjectUri := ref.UriFromString(assertion.Subject)
	if !subjectUri.HasType() {
		subjectUri = subjectUri.WithType("statement")
	}

	subject, _ := datastore.ActiveDataStore.FetchStatement(ctx, subjectUri)

	refs, _ := datastore.ActiveDataStore.FetchRefs(ctx, uri)
	enrichReferencesTo(ctx, &assertion, refs)

	data := struct {
		Uri         string
		ShortUri    string
		Assertion   assertions.Assertion
		IssuerLink  string
		IssuerName  string
		SubjectLink string
		SubjectText string
		ApiLink     string
		References  []ref.Reference
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

	uri := ref.MakeUri(key, "entity")
	entity, err := datastore.ActiveDataStore.FetchEntity(ctx, uri)
	if err != nil {
		HandleError(ctx, ErrorEntityFetch.instance("Error fetching entity "+uri.String()), w, r)
		return
	}

	refs, _ := datastore.ActiveDataStore.FetchRefs(ctx, entity.Uri())
	enrichReferencesTo(ctx, &entity, refs)

	data := struct {
		Uri        string
		ShortUri   string
		CommonName string
		ApiLink    string
		PublicKey  string
		References []ref.Reference
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

func NewStatementWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	username := authUsername(r)
	if username == "" {
		HandleError(ctx, ErrorNoAuth, w, r)
		return
	}
	user, err := datastore.ActiveDataStore.FetchUser(ctx, username)
	if err != nil {
		HandleError(ctx, ErrorUserNotFound.instance("User not found: "+username), w, r)
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

		keyUri := ref.UriFromString(keyId)

		if !user.HasKey(keyId) {
			HandleError(ctx, ErrorKeyAccess.instance("User does not have access to selected signing key"), w, r)
			return
		}

		assertion, err := datastore.CreateStatementAndAssertion(ctx, content, keyUri, assertions.IsTrue, confidence)
		if err != nil {
			HandleError(ctx, ErrorMakeAssertion.instance("Error making new statement and assertion"), w, r)
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

	results, _ := datastore.ActiveDataStore.Search(ctx, query)

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
	username := authUsername(r)
	if username == "" {
		HandleError(ctx, ErrorNoAuth, w, r)
		return
	}
	user, err := datastore.ActiveDataStore.FetchUser(ctx, username)
	if err != nil {
		HandleError(ctx, ErrorUserNotFound.instance("User not found when making new entity: "+username), w, r)
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
		entity := entities.Entity{CommonName: commonName}
		entity.MakeCertificate(privateKey)

		datastore.ActiveDataStore.Store(ctx, &entity)

		datastore.ActiveDataStore.StoreKey(entity.Uri(), entities.PrivateKeyToString(privateKey))

		user.AddKeyRef(entity.Uri().Escaped(), entity.CommonName)
		datastore.ActiveDataStore.StoreUser(ctx, user)

		// Redirect the user to the assertion
		http.Redirect(w, r, entity.Uri().WebPath(), http.StatusSeeOther)

		log.Infof("Redirecting to %s", entity.Uri().WebPath())
	}
}

func AddStatementAssertionWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	username := authUsername(r)
	if username == "" {
		HandleError(ctx, ErrorNoAuth, w, r)
		return
	}
	user, err := datastore.ActiveDataStore.FetchUser(ctx, username)
	if err != nil {
		HandleError(ctx, ErrorUserNotFound.instance("User not found when making new statement: "+username), w, r)
		return
	}

	statementHash := mux.Vars(r)["hash"]

	if r.Method == "GET" {
		statement, err := datastore.ActiveDataStore.FetchStatement(ctx, ref.MakeUri(statementHash, "statement"))
		if err != nil {
			log.Errorf("Error fetching statement: %v", err)
		} else {
			log.Debugf("Statement content = %s", statement.Content())
		}

		data := struct {
			Statement statements.Statement
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

		keyUri := ref.UriFromString(keyId)

		if !user.HasKey(keyId) {
			HandleError(ctx, ErrorKeyAccess.instance("User does not have access to selected signing key"), w, r)
			return
		}

		b64key, err := datastore.ActiveDataStore.FetchKey(keyUri)
		if err != nil {
			HandleError(ctx, ErrorKeyFetch.instance("Error fetching entity private key"), w, r)
			return
		}
		privateKey := entities.PrivateKeyFromString(b64key)

		entity, _ := datastore.ActiveDataStore.FetchEntity(ctx, keyUri)

		su := ref.MakeUri(statementHash, "statement")

		confidence, _ := strconv.ParseFloat(r.Form.Get("confidence"), 32)
		kind := r.Form.Get("assertion_type")

		assertion := datastore.CreateAssertion(ctx, su, entity.Uri(), assertions.AssertionTypeOf(kind), confidence, privateKey)

		// Redirect the user to the assertion
		http.Redirect(w, r, assertion.Uri().WebPath(), http.StatusSeeOther)

		log.DebugfX(ctx, "Redirecting to %s", assertion.Uri().WebPath())
	}
}
