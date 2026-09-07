package web

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestDocumentSchemaIsServed(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	resp, err := http.Get(wt.Server.URL + "/web/static/document.xsd")
	if err != nil {
		t.Fatalf("GET /web/static/document.xsd: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading schema: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(string(body), `element name="document"`) {
		t.Errorf("GET /web/static/document.xsd did not return the schema (len=%d)", len(body))
	}
}

func TestViewDoc(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	docs, _ := datastore.ActiveDataStore.Search(context.Background(), "GL93J73C")
	docHash := docs[0].Uri.Hash()

	page := wt.GetPage("/web/documents/" + docHash)
	page.AssertHtmlQuery("h2", "View Document")
	page.AssertHtmlQuery("h1", "About the Universe")
	page.AssertHtmlQuery("a", "The universe exists")
}

func TestSearchShowsDocumentType(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/search?query=GL93J73C")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h2", "Search results")
	got := page.Find(".searchresults tr td")
	if !strings.Contains(got, "Document") && !strings.Contains(got, "document") {
		t.Errorf("search result type = %q, want Document", got)
	}
}

func TestNewDocumentRequiresLogin(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	wt.AuthCookie = nil
	page := wt.GetPage("/web/newdocument")
	page.AssertErrorResponse()
	page.AssertHtmlQuery("#message", "Not logged in")
}

func TestNewDocumentPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/newdocument")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h2", "New Document")
	page.AssertHtmlQuery("label", "Document XML")
}

func TestPostNewDocument(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	docxml := `<?xml version="1.0" encoding="UTF-8"?>
<document>
	<metadata>
		<title>Web Test Document</title>
	</metadata>
	<section>
		<title>Section One</title>
		<paragraph>
			<span>Hello from the web test.</span>
		</paragraph>
	</section>
</document>`

	data := url.Values{
		"document": {docxml},
		"sign_as":  {user.KeyRefs[0].KeyId},
	}
	page := wt.PostFormData("/web/newdocument", data)
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h2", "View Document")
	page.AssertHtmlQuery("#title", "Web Test Document")

	docs, err := datastore.ActiveDataStore.Search(context.Background(), "Web Test Document")
	if err != nil {
		t.Fatalf("searching for new document: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("created document not found in datastore")
	}
	if _, err := datastore.ActiveDataStore.FetchDocument(context.Background(), docs[0].Uri); err != nil {
		t.Errorf("Error fetching new document: %v", err)
	}
}

func TestPostNewDocumentNotWellFormed(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	data := url.Values{
		"document": {"<document><metadata><title>Broken</title></metadata>"},
		"sign_as":  {user.KeyRefs[0].KeyId},
	}
	page := wt.PostFormData("/web/newdocument", data)
	page.AssertErrorResponse()
	page.AssertHtmlQuery("#message", "Error making document")

	docs, err := datastore.ActiveDataStore.Search(context.Background(), "Broken")
	if err != nil {
		t.Fatalf("searching after malformed document post: %v", err)
	}
	if len(docs) != 0 {
		t.Error("malformed document should not be stored")
	}
}
