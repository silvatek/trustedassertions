package main

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/web"
)

func setupTestServer() *httptest.Server {
	initLogging()

	web.TemplateDir = "../../web"
	testDataDir = "../../testdata"
	initDataStore()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := assertions.NewEntity("Signing entity", *big.NewInt(123456))
	signer.MakeCertificate(privateKey)
	datastore.ActiveDataStore.Store(&signer)
	datastore.ActiveDataStore.StoreKey(signer.Uri(), assertions.DecodePrivateKey(privateKey))
	web.DefaultEntityUri = signer.Uri()

	return httptest.NewServer(setupHandlers())
}

type WebPage struct {
	t            *testing.T
	url          string
	requestError error
	response     *http.Response
	statusCode   int
	htmlError    error
	html         *goquery.Document
}

func getWebPage(url string, t *testing.T) *WebPage {
	page := WebPage{url: url, t: t}
	page.response, page.requestError = http.Get(url)

	if page.requestError != nil {
		t.Errorf("Error fetching %s, %v", url, page.requestError)
		return &page
	}

	page.statusCode = page.response.StatusCode

	defer page.response.Body.Close()
	page.html, page.htmlError = goquery.NewDocumentFromReader(page.response.Body)

	return &page
}

func (page *WebPage) ok() bool {
	return (page.requestError == nil) && (page.statusCode < 400) && (page.htmlError == nil)
}

func (page *WebPage) assertHtmlQuery(query string, expected string) {
	if !page.ok() {
		return
	}
	results := page.html.Find(query)
	if !strings.Contains(results.Text(), expected) {
		page.t.Errorf("Did not find `%s` in [%s]", expected, query)
	}
}

func TestHomePage(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	page := getWebPage(server.URL+"/", t)
	page.assertHtmlQuery("h1", "Trusted Assertions")
}

func TestErrorPage(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	page := getWebPage(server.URL+"/web/broken", t)
	page.assertHtmlQuery("#intro", "Sorry, an error has occurred.")
	page.assertHtmlQuery("#message", "Fake error for testing")
}

func TestStatementPage(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	page := getWebPage(server.URL+"/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f", t)
	page.assertHtmlQuery("#content", "The universe exists")
}

func TestEntityPage(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	page := getWebPage(server.URL+"/web/entities/177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db", t)
	page.assertHtmlQuery("#common_name", "Mr Tester")
}

func TestAssertionPage(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	page := getWebPage(server.URL+"/web/assertions/514518bb09d57524bc6b96842721e4c4404cb4a3329aadf1761bb3eddb2832da", t)
	page.assertHtmlQuery("#category", "IsTrue")
}

func TestNewStatementPage(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	page := getWebPage(server.URL+"/web/newstatement", t)
	page.assertHtmlQuery("h1", "New Statement")
}

func TestPostNewStatement(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	data := url.Values{
		"statement": {"Test statement"},
	}

	response, err := http.Post(server.URL+"/web/newstatement", "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		t.Errorf("Error posting to %s : %v", "/web/newstatement", err)
	}
	defer response.Body.Close()
	html, _ := goquery.NewDocumentFromReader(response.Body)

	newUri := assertions.UriFromString(html.Find("#uri").Text())

	// Make sure the new assertion is really in the datastore
	_, err = datastore.ActiveDataStore.FetchAssertion(newUri)
	if err != nil {
		t.Errorf("Error fetching new assertion: %v", err)
	}
}
