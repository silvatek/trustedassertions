package webtest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/testcontext"
)

func MockHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("cookie") != "" {
		expiration := time.Now().Add(5 * time.Minute)
		cookie := &http.Cookie{Name: "testcookie", Path: "/", Value: "test", Expires: expiration, MaxAge: 0, SameSite: http.SameSiteStrictMode}
		http.SetCookie(w, cookie)
	}
	if r.URL.Query().Get("responsecode") != "" {
		code, _ := strconv.Atoi(r.URL.Query().Get("responsecode"))
		w.WriteHeader(code)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte("<html><body><h1>Test Heading</h1></body></html>"))
}

func MockEscapedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if r.URL.Query().Get("elsewhere") == "true" {
		w.Write([]byte(`<html><body><div id="other">&#34; onfocus=&#34;alert(1)&#34; x=&#34;</div><span id="searchterm">plain</span></body></html>`))
		return
	}
	w.Write([]byte(`<html><body><span id="searchterm">&#34; onfocus=&#34;alert(1)&#34; x=&#34;</span></body></html>`))
}

func MockHtmxHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Vary", "HX-Request")
	w.WriteHeader(http.StatusOK)
	rawBoost := `hx-boost="false"`
	if r.URL.Query().Get("boostedapi") == "true" {
		rawBoost = ""
	}
	if r.URL.Query().Get("fragment") == "true" {
		w.Write([]byte(`<div id="pagemenu" hx-swap-oob="true"><a href="/api/v1/statements/abc" ` + rawBoost + `>Raw</a></div><div id="page">content</div>`))
		return
	}
	w.Write([]byte(`<!doctype html><html><head><script src="https://cdn.jsdelivr.net/npm/htmx.org@2.0.10/dist/htmx.min.js"></script></head><body hx-boost="true" hx-target="#page" hx-swap="outerHTML"><div id="pagemenu" hx-swap-oob="true"><a href="/api/v1/statements/abc" ` + rawBoost + `>Raw</a></div><div id="page">content</div></body></html>`))
}

func SetupHtmxTestServer(wt *WebTest) {
	router := mux.NewRouter()
	router.HandleFunc("/", MockHtmxHandler)
	wt.Server = httptest.NewServer(router)
}

func SetupTestServer(wt *WebTest) {
	router := mux.NewRouter()
	router.HandleFunc("/", MockHandler)
	wt.Server = httptest.NewServer(router)
}

func TestWebTesterHappyPath(t *testing.T) {
	wt := MakeWebTest(t)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/")

	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h1", "Test Heading")
	page.AssertNoCookie("auth")

	if page.Text() == "" {
		t.Error("Unexpected empty page")
	}

	data := url.Values{
		"testing": {"Testing"},
	}
	page = wt.PostFormData("/", data)
	page.AssertSuccessResponse()
}

func TestRequestError(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)

	wt.Server = &httptest.Server{}
	wt.Server.URL = "http://localhost:0/"

	wt.GetPage("/")
	if !t1.ErrorsFound {
		t.Error("No errors reported to MockTestContext")
	}

	t1.ErrorsFound = false
	wt.PostFormData("/", url.Values{})

	t1.AssertErrorsFound(t)
}

func TestExpectedErrorResponse(t *testing.T) {
	wt := MakeWebTest(t)
	defer wt.Close()

	router := mux.NewRouter()
	wt.Server = httptest.NewServer(router)

	page := wt.GetPage("/")
	page.AssertErrorResponse()
}

func TestUnexpectedErrorResponse(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/broken")
	page.AssertSuccessResponse()

	t1.AssertErrorsFound(t)
}

func TestUnexpectedSuccessResponse(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertErrorResponse()

	t1.AssertErrorsFound(t)
}

func TestCookiePresent(t *testing.T) {
	wt := MakeWebTest(t)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/?cookie=true")
	page.AssertSuccessResponse()
	page.AssertHasCookie("testcookie")
}

func TestCookieIncorrectlyFound(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/?cookie=true")
	page.AssertSuccessResponse()
	page.AssertNoCookie("testcookie")

	t1.AssertErrorsFound(t)
}

func TestExpectedCookieNotFound(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertSuccessResponse()
	page.AssertHasCookie("testcookie")

	t1.AssertErrorsFound(t)
}

func TestAuthCookie(t *testing.T) {
	wt := MakeWebTest(t)
	wt.AuthCookie = MakeAuthCookie("tester")
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertSuccessResponse()

	page = wt.PostFormData("/", url.Values{})
	page.AssertSuccessResponse()
}

func MakeAuthCookie(userId string) *http.Cookie {
	expiration := time.Now().Add(2 * time.Hour)
	cookie := http.Cookie{Name: "auth", Path: "/", Value: userId, Expires: expiration, SameSite: http.SameSiteStrictMode}
	return &cookie
}

func TestBrokenPage(t *testing.T) {
	wtc := testcontext.MockTestContext{}
	wt := MakeWebTest(&wtc)
	page := WebPage{wt: wt, htmlError: errors.New("request failed")}

	if page.Find("anything") != "" {
		t.Error("Find succeeded in broken page")
	}
	page.AssertNoCookie("testing")
	page.AssertHasCookie("testing")
	page.AssertHtmlQuery("h1", "Some heading")
}

func TestErrorSummary(t *testing.T) {
	page := WebPage{}
	if page.errorSummary() != "" {
		t.Errorf("Unexpected non-blank error summary: %s", page.errorSummary())
	}

	page.htmlError = errors.New("err123")
	if page.errorSummary() != "HTML error: err123" {
		t.Errorf("Unexpected HTML error summary: %s", page.errorSummary())
	}

	page.statusCode = 500
	if page.errorSummary() != "Error response code: 500" {
		t.Errorf("Unexpected status code error summary: %s", page.errorSummary())
	}

	page.requestError = errors.New("err789")
	if page.errorSummary() != "Request error: err789" {
		t.Errorf("Unexpected request error summary: %s", page.errorSummary())
	}
}

func TestPostErrorStatus(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.PostFormData("/?responsecode=500", url.Values{})
	page.AssertSuccessResponse()

	t1.AssertErrorsFound(t)
}

func TestElementNotfound(t *testing.T) {
	wtc := testcontext.MockTestContext{}
	wt := MakeWebTest(&wtc)
	SetupTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h1", "**TextNotInPage**")

	wtc.AssertErrorsFound(t)
}

func TestAssertHtmxFullPage(t *testing.T) {
	wt := MakeWebTest(t)
	SetupHtmxTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertSuccessResponse()
	page.AssertHtmxFragment(false)
}

func TestAssertHtmxFragment(t *testing.T) {
	wt := MakeWebTest(t)
	SetupHtmxTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/?fragment=true")
	page.AssertSuccessResponse()
	page.AssertHtmxFragment(true)
}

func TestAssertHtmxFragmentMismatch(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupHtmxTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertSuccessResponse()
	page.AssertHtmxFragment(true)

	t1.AssertErrorsFound(t)
}

func TestAssertHtmxFullPageMismatch(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupHtmxTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/?fragment=true")
	page.AssertSuccessResponse()
	page.AssertHtmxFragment(false)

	t1.AssertErrorsFound(t)
}

func TestAssertHtmxFragmentBrokenPage(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	page := WebPage{wt: wt, htmlError: errors.New("request failed")}
	page.AssertHtmxFragment(false)
	t1.AssertErrorsFound(t)
}

func TestAssertHtmxBoostedApiLink(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	SetupHtmxTestServer(wt)
	defer wt.Close()

	page := wt.GetPage("/?boostedapi=true")
	page.AssertSuccessResponse()
	page.AssertHtmxFragment(false)

	t1.AssertErrorsFound(t)
}

func TestAssertHtmlQueryEscaped(t *testing.T) {
	wt := MakeWebTest(t)
	router := mux.NewRouter()
	router.HandleFunc("/", MockEscapedHandler)
	wt.Server = httptest.NewServer(router)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertSuccessResponse()
	page.AssertHtmlQueryEscaped("#searchterm", `" onfocus="alert(1)" x="`)
}

func TestAssertHtmlQueryEscapedOnlyInQuery(t *testing.T) {
	t1 := testcontext.MockTestContext{}
	wt := MakeWebTest(&t1)
	router := mux.NewRouter()
	router.HandleFunc("/", MockEscapedHandler)
	wt.Server = httptest.NewServer(router)
	defer wt.Close()

	page := wt.GetPage("/?elsewhere=true")
	page.AssertSuccessResponse()
	page.AssertHtmlQueryEscaped("#searchterm", `" onfocus="alert(1)" x="`)

	t1.AssertErrorsFound(t)
}
