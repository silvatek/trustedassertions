package webtest

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
)

func MockHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<html><body><h1>Test Heading</h1></body></html>"))
}

type MockTestContext struct {
	ErrorsFound bool
}

func (t *MockTestContext) Error(args ...any) {
	t.ErrorsFound = true
}

func (t *MockTestContext) Errorf(format string, args ...any) {
	t.ErrorsFound = true
}

func TestWebTesterHappyPath(t *testing.T) {
	wt := MakeWebTest(t)
	defer wt.Close()

	router := mux.NewRouter()
	router.HandleFunc("/", MockHandler)
	wt.Server = httptest.NewServer(router)

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
	t1 := MockTestContext{}
	wt := MakeWebTest(&t1)

	wt.Server = &httptest.Server{}
	wt.Server.URL = "http://localhost:0/"

	wt.GetPage("/")
	if !t1.ErrorsFound {
		t.Error("No errors reported to MockTestContext")
	}
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
	t1 := MockTestContext{}
	wt := MakeWebTest(&t1)
	defer wt.Close()

	router := mux.NewRouter()
	router.HandleFunc("/", MockHandler)
	wt.Server = httptest.NewServer(router)

	page := wt.GetPage("/broken")
	page.AssertSuccessResponse()

	if !t1.ErrorsFound {
		t.Error("No errors reported to MockTestContext")
	}
}

func TestUnexpectedSuccessResponse(t *testing.T) {
	t1 := MockTestContext{}
	wt := MakeWebTest(&t1)
	defer wt.Close()

	router := mux.NewRouter()
	router.HandleFunc("/", MockHandler)
	wt.Server = httptest.NewServer(router)

	page := wt.GetPage("/")
	page.AssertErrorResponse()

	if !t1.ErrorsFound {
		t.Error("No errors reported to MockTestContext")
	}
}

func TestFindInBrokenPage(t *testing.T) {
	page := WebPage{htmlError: errors.New("request failed")}
	if page.Find("anything") != "" {
		t.Error("Find succeeded in broken page")
	}
}
