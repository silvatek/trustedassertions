package web

import (
	"net/http"
	"testing"

	"silvatek.uk/trustedassertions/internal/webtest"
)

const missingHash = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

func assertNotFoundPage(t *testing.T, page *webtest.WebPage) {
	t.Helper()
	if page.Status() != http.StatusNotFound {
		t.Errorf("status = %d, want 404", page.Status())
	}
	page.AssertHtmlQuery("h2", "Page not found")
	page.AssertHtmlQuery("#intro", "Sorry, we couldn't find that page.")
}

func TestUnknownRouteIsNotFound(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	assertNotFoundPage(t, wt.GetPage("/web/does-not-exist"))
}

func TestMissingStatementIsNotFound(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	assertNotFoundPage(t, wt.GetPage("/web/statements/"+missingHash))
}

func TestMissingEntityIsNotFound(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	assertNotFoundPage(t, wt.GetPage("/web/entities/"+missingHash))
}

func TestMissingAssertionIsNotFound(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	assertNotFoundPage(t, wt.GetPage("/web/assertions/"+missingHash))
}

func TestMissingDocumentIsNotFound(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	assertNotFoundPage(t, wt.GetPage("/web/documents/"+missingHash))
}

func TestAddAssertionMissingStatementIsNotFound(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	assertNotFoundPage(t, wt.GetPage("/web/statements/"+missingHash+"/addassertion"))
}
