package web

import (
	"net/http"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/docs"
	ref "silvatek.uk/trustedassertions/internal/references"
)

func ViewDocumentWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	key := mux.Vars(r)["hash"]
	document, _ := datastore.ActiveDataStore.FetchDocument(ctx, ref.MakeUri(key, "document"))

	data := struct {
		Doc       docs.Document
		Title     string
		DocHtml   string
		AuthorUri ref.HashUri
	}{
		Doc:       document,
		Title:     document.Summary(),
		DocHtml:   document.ToHtml(),
		AuthorUri: ref.UriFromString(document.Metadata.Author.Entity),
	}

	RenderWebPage(ctx, "viewdocument", data, nil, w, r)
}

func NewDocumentWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)

	username := authUsername(r)
	if username == "" {
		HandleError(ctx, ErrorNoAuth, w, r)
		return
	}
	user, err := datastore.ActiveDataStore.FetchUser(ctx, username)
	if err != nil {
		HandleError(ctx, ErrorUserNotFound.instance("User not found when making new document: "+username), w, r)
		return
	}

	if r.Method == "GET" {
		data := struct {
			User auth.User
		}{
			User: user,
		}

		RenderWebPage(ctx, "newdocumentform", data, nil, w, r)
	} else if r.Method == "POST" {
		log.InfofX(ctx, "Creating new document")
		r.ParseForm()

		keyId := r.Form.Get("sign_as")
		if !user.HasKey(keyId) {
			HandleError(ctx, ErrorKeyAccess.instance("User does not have access to selected signing key"), w, r)
			return
		}

		keyUri := ref.UriFromString(keyId)

		docxml := r.Form.Get("document")

		doc, err := datastore.CreateDocumentAndAssertions(ctx, docxml, keyUri)
		if err != nil {
			HandleError(ctx, ErrorMakeDocument.instance("Error creating new document"), w, r)
			return
		}

		http.Redirect(w, r, doc.Uri().WebPath(), http.StatusSeeOther)
	}

}
