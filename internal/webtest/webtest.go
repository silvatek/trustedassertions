package webtest

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"silvatek.uk/trustedassertions/internal/testcontext"
	// "silvatek.uk/trustedassertions/internal/auth"
)

type WebTest struct {
	t          testcontext.TestContext
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
	body         []byte
}

func MakeWebTest(t testcontext.TestContext) *WebTest {
	wt := WebTest{t: t}

	jar, _ := cookiejar.New(nil)
	wt.Client = &http.Client{
		Jar: jar,
	}

	return &wt
}

func (wt *WebTest) PostJSON(path string, body []byte) *WebPage {
	reqURL := wt.Server.URL + path
	req, _ := http.NewRequest("POST", reqURL, bytes.NewReader(body))
	if wt.AuthCookie != nil {
		req.AddCookie(wt.AuthCookie)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	page := WebPage{url: reqURL, wt: wt}
	response, err := wt.Client.Do(req)
	if err != nil {
		page.requestError = err
		wt.t.Errorf("Error posting %s, %v", reqURL, page.requestError)
		return &page
	}
	page.response = response
	page.statusCode = response.StatusCode
	defer page.response.Body.Close()
	page.body, page.htmlError = io.ReadAll(page.response.Body)
	return &page
}

func (page *WebPage) Status() int {
	return page.statusCode
}

func (page *WebPage) RawBody() []byte {
	return page.body
}

func (wt *WebTest) GetPage(path string) *WebPage {
	return wt.GetPageWithHeaders(path, nil)
}

func (wt *WebTest) GetPageWithHeaders(path string, headers map[string]string) *WebPage {
	url := wt.Server.URL + path
	page := WebPage{url: url, wt: wt}

	req, _ := http.NewRequest("GET", url, nil)
	if wt.AuthCookie != nil {
		req.AddCookie(wt.AuthCookie)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	page.response, page.requestError = wt.Client.Do(req)

	if page.requestError != nil {
		wt.t.Errorf("Error fetching %s, %v", url, page.requestError)
		return &page
	}

	page.statusCode = page.response.StatusCode

	defer page.response.Body.Close()
	page.body, page.htmlError = io.ReadAll(page.response.Body)
	if page.htmlError != nil {
		return &page
	}
	page.html, page.htmlError = goquery.NewDocumentFromReader(bytes.NewReader(page.body))

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
		wt.t.Errorf("Error posting %s, %v", url, page.requestError)
		return &page
	}

	page.response = response
	page.statusCode = response.StatusCode
	if page.statusCode >= 400 {
		return &page
	}

	defer page.response.Body.Close()
	body, err := io.ReadAll(page.response.Body)
	if err != nil {
		page.htmlError = err
		return &page
	}
	page.body = body
	page.html, page.htmlError = goquery.NewDocumentFromReader(bytes.NewReader(page.body))

	return &page
}

func (page *WebPage) ok() bool {
	return (page.requestError == nil) && (page.statusCode < 400) && (page.htmlError == nil)
}

func (page *WebPage) errorSummary() string {
	if page.requestError != nil {
		return fmt.Sprintf("Request error: %v", page.requestError)
	}
	if page.statusCode >= 400 {
		return fmt.Sprintf("Error response code: %d", page.statusCode)
	}
	if page.htmlError != nil {
		return fmt.Sprintf("HTML error: %v", page.htmlError)
	}
	return ""
}

func (page *WebPage) Find(q string) string {
	if !page.ok() {
		return ""
	}
	return page.html.Find(q).Text()
}

func (page *WebPage) AssertSuccessResponse() {
	if !page.ok() {
		page.wt.t.Error(page.errorSummary())
	}
}

func (page *WebPage) AssertErrorResponse() {
	if page.ok() {
		page.wt.t.Error("Unexpected success response")
	}
}

func (page *WebPage) AssertHtmlQuery(query string, expected string) {
	if page.html == nil {
		if !page.ok() {
			page.wt.t.Error(page.errorSummary())
		}
		return
	}
	results := page.html.Find(query).Text()
	if !strings.Contains(results, expected) {
		page.wt.t.Errorf("Did not find `%s` in [%s]", expected, query)
	}
}

func (page *WebPage) AssertHtmlQueryEscaped(query string, expected string) {
	if !page.ok() {
		page.wt.t.Error(page.errorSummary())
		return
	}
	sel := page.html.Find(query)
	if sel.Length() == 0 {
		page.wt.t.Errorf("Did not find [%s]", query)
		return
	}
	fragment, err := goquery.OuterHtml(sel)
	if err != nil {
		page.wt.t.Errorf("Error reading HTML for [%s]: %v", query, err)
		return
	}
	escaped := html.EscapeString(expected)
	if !strings.Contains(fragment, escaped) {
		page.wt.t.Errorf("Did not find escaped `%s` in [%s]", escaped, query)
	}
	if escaped != expected && strings.Contains(fragment, expected) {
		page.wt.t.Errorf("Unescaped `%s` found in [%s]", expected, query)
	}
	page.AssertHtmlQuery(query, expected)
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

func (page *WebPage) Attr(query string, name string) string {
	if !page.ok() {
		return ""
	}
	value, exists := page.html.Find(query).Attr(name)
	if !exists {
		return ""
	}
	return value
}

func (page *WebPage) Header(name string) string {
	if page.response == nil {
		return ""
	}
	return page.response.Header.Get(name)
}

// AssertHtmxFragment checks that the response is a well-formed HTMX fragment
// when fragment is true, or a well-formed full page (with the HTMX library and
// boost attributes) when fragment is false. Both forms must send Vary: HX-Request
// and must disable boosting on any /api/ links.
func (page *WebPage) AssertHtmxFragment(fragment bool) {
	if !page.ok() {
		page.wt.t.Error(page.errorSummary())
		return
	}
	if !strings.Contains(page.Header("Vary"), "HX-Request") {
		page.wt.t.Errorf("Expected Vary: HX-Request, got %s", page.Header("Vary"))
	}
	if fragment {
		page.assertHtmxFragment()
	} else {
		page.assertHtmxFullPage()
	}
	page.assertApiLinksNotBoosted()
}

func (page *WebPage) assertHtmxFragment() {
	if page.html.Find("#page").Length() == 0 {
		page.wt.t.Error("HTMX fragment should include #page")
	}
	if page.Attr("#pagemenu", "hx-swap-oob") != "true" {
		page.wt.t.Error("HTMX fragment should include an out-of-band pagemenu swap")
	}
	if page.Attr("body", "hx-boost") == "true" {
		page.wt.t.Error("HTMX fragment should not include the full page shell")
	}
	if strings.Contains(page.Find("script"), "htmx.org") {
		page.wt.t.Error("HTMX fragment should not reload the htmx library")
	}
}

func (page *WebPage) assertHtmxFullPage() {
	if page.Attr("body", "hx-boost") != "true" {
		page.wt.t.Error("Expected hx-boost on body")
	}
	if page.Attr("body", "hx-target") != "#page" {
		page.wt.t.Errorf("Expected hx-target of #page, got %s", page.Attr("body", "hx-target"))
	}
	if page.Attr("body", "hx-swap") != "outerHTML" {
		page.wt.t.Errorf("Expected hx-swap of outerHTML, got %s", page.Attr("body", "hx-swap"))
	}
	src := page.Attr("script[src*='htmx']", "src")
	if !strings.Contains(src, "htmx.org") {
		page.wt.t.Errorf("Expected htmx script, got src %s", src)
	}
	if page.html.Find("#page").Length() == 0 {
		page.wt.t.Error("Full page should include #page")
	}
}

func (page *WebPage) assertApiLinksNotBoosted() {
	links := page.html.Find(`a[href^="/api/"]`)
	if links.Length() == 0 {
		return
	}
	links.Each(func(_ int, s *goquery.Selection) {
		boost, _ := s.Attr("hx-boost")
		if boost != "false" {
			href, _ := s.Attr("href")
			page.wt.t.Errorf("API/raw link %s should disable htmx boosting, hx-boost=%q", href, boost)
		}
	})
}
