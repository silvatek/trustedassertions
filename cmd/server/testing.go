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
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/web"
)

type WebTest struct {
	t      *testing.T
	user   *auth.User
	server *httptest.Server
}

func NewWebTest(t *testing.T) *WebTest {
	wt := WebTest{t: t}
	wt.setupTestServer()
	return &wt
}

func (wt *WebTest) Close() {
	wt.server.Close()
}

func (wt *WebTest) setupTestServer() {
	initLogging()

	web.TemplateDir = "../../web"
	testDataDir = "../../testdata"
	initDataStore()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := assertions.NewEntity("Signing entity", *big.NewInt(123456))
	signer.MakeCertificate(privateKey)
	datastore.ActiveDataStore.Store(&signer)
	datastore.ActiveDataStore.StoreKey(signer.Uri(), assertions.PrivateKeyToString(privateKey))
	web.DefaultEntityUri = signer.Uri()

	wt.user = &auth.User{Id: "admin"}
	wt.user.KeyRefs = append(wt.user.KeyRefs, auth.KeyRef{UserId: wt.user.Id, KeyId: signer.Uri().Unadorned(), Summary: ""})
	datastore.ActiveDataStore.StoreUser(*wt.user)

	wt.server = httptest.NewServer(setupHandlers())
}

type WebPage struct {
	wt           *WebTest
	url          string
	requestError error
	response     *http.Response
	statusCode   int
	htmlError    error
	html         *goquery.Document
}

func (wt *WebTest) getPage(path string) *WebPage {
	url := wt.server.URL + path
	page := WebPage{url: url, wt: wt}
	page.response, page.requestError = http.Get(url)

	if page.requestError != nil {
		wt.t.Errorf("Error fetching %s, %v", url, page.requestError)
		return &page
	}

	page.statusCode = page.response.StatusCode

	defer page.response.Body.Close()
	page.html, page.htmlError = goquery.NewDocumentFromReader(page.response.Body)

	return &page
}

func postFormData(url string, data url.Values, user *auth.User) (*http.Response, error) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if user != nil {
		cookie, _ := web.MakeAuthCookie(*user)
		req.AddCookie(&cookie)
	}

	return http.DefaultClient.Do(req)
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
		page.wt.t.Errorf("Did not find `%s` in [%s]", expected, query)
	}
}
