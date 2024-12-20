package webtest

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	// "silvatek.uk/trustedassertions/internal/auth"
)

// TestContext is compatible with testing.T but can also be mocked.
type TestContext interface {
	Error(args ...any)
	Errorf(format string, args ...any)
}

type WebTest struct {
	t          TestContext
	Server     *httptest.Server
	Passwd     string
	AuthCookie *http.Cookie
	Client     *http.Client
}

func (wt *WebTest) Close() {
	if wt.Server != nil && wt.Server.Config != nil {
		wt.Server.Close()
	}
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

func MakeWebTest(t TestContext) *WebTest {
	wt := WebTest{t: t}

	jar, _ := cookiejar.New(nil)
	wt.Client = &http.Client{
		Jar: jar,
	}

	return &wt
}

func (wt *WebTest) GetPage(path string) *WebPage {
	url := wt.Server.URL + path
	page := WebPage{url: url, wt: wt}

	req, _ := http.NewRequest("GET", url, nil)
	if wt.AuthCookie != nil {
		req.AddCookie(wt.AuthCookie)
	}

	page.response, page.requestError = wt.Client.Do(req)

	if page.requestError != nil {
		wt.t.Errorf("Error fetching %s, %v", url, page.requestError)
		return &page
	}

	page.statusCode = page.response.StatusCode

	defer page.response.Body.Close()
	page.html, page.htmlError = goquery.NewDocumentFromReader(page.response.Body)

	return &page
}

func (wt *WebTest) PostFormData(path string, data url.Values) *WebPage {
	url := wt.Server.URL + path
	req, _ := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if wt.AuthCookie != nil {
		req.AddCookie(wt.AuthCookie)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	page := WebPage{url: url, wt: wt}

	response, err := wt.Client.Do(req)
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

func (page *WebPage) AssertSuccessResponse() {
	if page.statusCode >= 400 {
		page.wt.t.Errorf("Response code indicates error: %d", page.statusCode)
	}
}

func (page *WebPage) AssertErrorResponse() {
	if page.statusCode < 400 {
		page.wt.t.Errorf("Response code does not indicate error: %d", page.statusCode)
	}
}

func (page *WebPage) AssertHtmlQuery(query string, expected string) {
	if !page.ok() {
		return
	}
	results := page.Find(query)
	if !strings.Contains(results, expected) {
		page.wt.t.Errorf("Did not find `%s` in [%s]", expected, query)
	}
}

func (page *WebPage) AssertHasCookie(name string) {
	if !page.ok() {
		return
	}
	url, _ := url.Parse(page.wt.Server.URL + "/")

	for _, cookie := range page.wt.Client.Jar.Cookies(url) {
		if cookie.Name == name {
			return // cookie found, no error
		}
	}

	page.wt.t.Errorf("`%s` cookie not found", name)
}

func (page *WebPage) AssertNoCookie(name string) {
	if !page.ok() {
		return
	}
	url, _ := url.Parse(page.wt.Server.URL + "/")

	for _, cookie := range page.wt.Client.Jar.Cookies(url) {
		if cookie.Name == name {
			if cookie.Value != "" {
				page.wt.t.Errorf("`%s` cookie found", name)
			}
		}
	}
}

func (page *WebPage) Text() string {
	return page.html.Text()
}
