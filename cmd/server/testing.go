package main

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"net/http"
	"net/http/cookiejar"
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
	passwd string
	client *http.Client
}

func NewWebTest(t *testing.T) *WebTest {
	wt := WebTest{t: t}
	wt.setupTestServer()

	jar, _ := cookiejar.New(nil)
	wt.client = &http.Client{
		Jar: jar,
	}

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
	wt.passwd = "testing"
	wt.user.HashPassword(wt.passwd)
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

	req, _ := http.NewRequest("GET", url, nil)
	addAuthCookie(wt.user, req)

	page.response, page.requestError = wt.client.Do(req)

	if page.requestError != nil {
		wt.t.Errorf("Error fetching %s, %v", url, page.requestError)
		return &page
	}

	page.statusCode = page.response.StatusCode

	defer page.response.Body.Close()
	page.html, page.htmlError = goquery.NewDocumentFromReader(page.response.Body)

	return &page
}

func addAuthCookie(user *auth.User, req *http.Request) {
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	if user != nil {
		cookie, _ := web.MakeAuthCookie(*user)
		req.AddCookie(&cookie)
	}
}

func (wt *WebTest) postFormData(path string, data url.Values) *WebPage {
	url := wt.server.URL + path
	req, _ := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	addAuthCookie(wt.user, req)

	page := WebPage{url: url, wt: wt}

	response, err := wt.client.Do(req)
	if err != nil {
		page.requestError = err
		return &page
	}

	page.response = response
	page.statusCode = response.StatusCode
	if page.statusCode >= 400 {
		return &page
	}

	defer page.response.Body.Close()
	page.html, page.htmlError = goquery.NewDocumentFromReader(page.response.Body)

	return &page
}

func (page *WebPage) ok() bool {
	return (page.requestError == nil) && (page.statusCode < 400) && (page.htmlError == nil)
}

func (page *WebPage) Find(q string) string {
	if !page.ok() {
		return ""
	}
	return page.html.Find(q).Text()
}

func (page *WebPage) assertSuccessResponse() {
	if page.statusCode >= 400 {
		page.wt.t.Errorf("Response code indicates error: %d", page.statusCode)
	}
}

func (page *WebPage) assertErrorResponse() {
	if page.statusCode < 400 {
		page.wt.t.Errorf("Response code does not indicate error: %d", page.statusCode)
	}
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

func (page *WebPage) assertHasCookie(name string) {
	if !page.ok() {
		return
	}
	url, _ := url.Parse(page.wt.server.URL + "/")

	for _, cookie := range page.wt.client.Jar.Cookies(url) {
		if cookie.Name == name {
			return // cookie found, no error
		}
	}

	page.wt.t.Errorf("`%s` cookie not found", name)
}

func (page *WebPage) assertNoCookie(name string) {
	if !page.ok() {
		return
	}
	url, _ := url.Parse(page.wt.server.URL + "/")

	for _, cookie := range page.wt.client.Jar.Cookies(url) {
		if cookie.Name == name {
			page.wt.t.Errorf("`%s` cookie found", name)
		}
	}

}
