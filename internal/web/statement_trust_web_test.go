package web

import (
	"context"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/entities"
	. "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/webtest"
)

const testStatementHash = "e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f"

func testStatementPath() string {
	return "/web/statements/" + testStatementHash
}

func testStatementTrustPath() string {
	return testStatementPath() + "/trust"
}

func testStatementUri() HashUri {
	return MakeUri(testStatementHash, "statement")
}

func TestStatementPageTrustHiddenWhenLoggedOut(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	wt.AuthCookie = nil
	page := wt.GetPage(testStatementPath())
	page.AssertSuccessResponse()
	if strings.Contains(string(page.RawBody()), `id="statement-trust"`) {
		t.Error("expected no trust placeholder when logged out")
	}
	if strings.Contains(page.Find(".fieldset"), "Veracity") {
		t.Error("expected no Veracity field when logged out")
	}
}

func TestStatementPageHasTrustPlaceholderWhenLoggedIn(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage(testStatementPath())
	page.AssertSuccessResponse()
	page.AssertHtmlQuery(".fieldset", "Veracity")
	if got := page.Attr("#statement-trust", "hx-get"); got != testStatementTrustPath() {
		t.Errorf("hx-get = %q, want %q", got, testStatementTrustPath())
	}
	if got := page.Attr("#statement-trust", "hx-trigger"); got != "load" {
		t.Errorf("hx-trigger = %q, want load", got)
	}
	if got := page.Attr("#statement-trust", "hx-target"); got != "this" {
		t.Errorf("hx-target = %q, want this", got)
	}
	if got := page.Attr("#statement-trust", "hx-swap"); got != "innerHTML" {
		t.Errorf("hx-swap = %q, want innerHTML", got)
	}
	if page.Find("#statement-trust-score") != "" {
		t.Error("statement page should not include a trust score before the fragment loads")
	}
	if strings.Contains(page.Find("#statement-trust"), "no matching assertions") {
		t.Error("statement page should not include the trust fragment yet")
	}
}

func TestStatementTrustFragmentRequiresLogin(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	wt.AuthCookie = nil
	page := wt.GetPage(testStatementTrustPath())
	page.AssertErrorResponse()
	page.AssertHtmlQuery("#message", "Not logged in")
}

func TestStatementTrustFragmentEmptyTrust(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage(testStatementTrustPath())
	page.AssertSuccessResponse()
	assertTrustFragment(t, page)
	page.AssertHtmlQuery("#statement-trust-empty", "Unable to verify trustworthiness of this statement")
	if strings.Contains(strings.ToLower(string(page.RawBody())), "root") {
		t.Error("empty-trust copy should not say root")
	}
	if page.Find("#statement-trust-score") != "" {
		t.Error("expected no score when the user has not said they trust anyone")
	}
}

func TestStatementTrustFragmentUndefined(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	trustDefaultEntity(t, 0.75)

	page := wt.GetPage(testStatementTrustPath())
	page.AssertSuccessResponse()
	assertTrustFragment(t, page)
	page.AssertHtmlQuery("#statement-trust-undefined", "no matching assertions")
	if page.Find("#statement-trust-score") != "" {
		t.Error("ErrUndefined should not display a numeric score")
	}
}

func TestStatementTrustFragmentScore(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	trustDefaultEntity(t, 0.75)
	addAssertionOnTestStatement(t, assertions.IsTrue, 0.80)

	page := wt.GetPage(testStatementTrustPath())
	page.AssertSuccessResponse()
	assertTrustFragment(t, page)
	page.AssertHtmlQuery("#statement-trust-score", "0.60")
	if page.Find("#statement-trust-undefined") != "" {
		t.Error("defined score should not use the undefined message")
	}
	if strings.Contains(string(page.RawBody()), "Estimated chance") {
		t.Error("score fragment should be the value only")
	}
}

func TestStatementTrustFragmentCancelled(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	trustDefaultEntity(t, 0.75)
	addAssertionOnTestStatement(t, assertions.IsTrue, 0.80)
	addAssertionOnTestStatement(t, assertions.IsFalse, 0.80)

	page := wt.GetPage(testStatementTrustPath())
	page.AssertSuccessResponse()
	assertTrustFragment(t, page)
	page.AssertHtmlQuery("#statement-trust-score", "0.50")
	if page.Find("#statement-trust-undefined") != "" {
		t.Error("cancelled 0.50 should be a displayed score, not ErrUndefined")
	}
}

func TestStatementTrustFragmentUnknownStatement(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	assertNotFoundPage(t, wt.GetPage("/web/statements/"+missingHash+"/trust"))
}

func assertTrustFragment(t *testing.T, page *webtest.WebPage) {
	t.Helper()
	body := string(page.RawBody())
	if strings.Contains(body, `id="page"`) {
		t.Error("trust fragment should not include #page")
	}
	if strings.Contains(body, `id="pagemenu"`) {
		t.Error("trust fragment should not include the menu")
	}
	if strings.Contains(body, "htmx.org") {
		t.Error("trust fragment should not reload the htmx library")
	}
}

func trustDefaultEntity(t *testing.T, level float64) {
	t.Helper()
	stored, err := datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	stored.AddTrustRoot(DefaultEntityUri, level)
	datastore.ActiveDataStore.StoreUser(context.TODO(), stored)
}

func addAssertionOnTestStatement(t *testing.T, kind assertions.AssertionType, confidence float64) {
	t.Helper()
	key, err := datastore.ActiveDataStore.FetchKey(DefaultEntityUri)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := entities.PrivateKeyFromString(key)
	datastore.CreateAssertion(context.Background(), testStatementUri(), DefaultEntityUri, kind, confidence, privateKey)
}
