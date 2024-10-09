package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/web"
)

func setupTestServer() *httptest.Server {
	initLogging()

	web.TemplateDir = "../../web"
	testDataDir = "../../testdata"
	initDataStore()

	server := httptest.NewServer(setupHandlers())
	return server
}

type WebPage struct {
	t            *testing.T
	url          string
	requestError error
	response     *http.Response
	statusCode   int
	readError    error
	body         string
}

func getWebPage(url string, t *testing.T) *WebPage {
	page := WebPage{url: url, t: t}
	page.response, page.requestError = http.Get(url)

	if page.requestError != nil {
		return &page
	}

	page.statusCode = page.response.StatusCode
	var bytes []byte
	bytes, page.readError = io.ReadAll(page.response.Body)

	if page.readError != nil {
		return &page
	}

	page.body = string(bytes)

	return &page
}

func (page *WebPage) assertHtmlContains(s string) {
	if !strings.Contains(page.body, s) {
		page.t.Errorf("Did not find %s in %s", s, page.body)
	}
}

func TestWebApp(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	page := getWebPage(server.URL+"/", t)
	page.assertHtmlContains("Trusted Assertions")

	page = getWebPage(server.URL+"/web/broken", t)
	page.assertHtmlContains("Sorry, an error has occurred.")
	page.assertHtmlContains("Fake error for testing")

	page = getWebPage(server.URL+"/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f", t)
	page.assertHtmlContains("The universe exists")

	page = getWebPage(server.URL+"/web/entities/177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db", t)
	page.assertHtmlContains("Mr Tester")

	page = getWebPage(server.URL+"/web/assertions/514518bb09d57524bc6b96842721e4c4404cb4a3329aadf1761bb3eddb2832da", t)
	page.assertHtmlContains("IsTrue")
}
