package web

import (
	"context"
	"net/url"
	"testing"

	"silvatek.uk/trustedassertions/internal/datastore"
	. "silvatek.uk/trustedassertions/internal/references"
)

const testEntityHash = "177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db"

func testEntityPath() string {
	return "/web/entities/" + testEntityHash
}

func testEntityUri() HashUri {
	return MakeUri(testEntityHash, "entity")
}

func TestEntityPageTrustFormWhenLoggedIn(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage(testEntityPath())
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#trust-form", "I trust this entity")
	page.AssertHtmlQuery("#trust-form", "slightly (20%)")
	page.AssertHtmlQuery("#trust-form", "somewhat (50%)")
	page.AssertHtmlQuery("#trust-form", "a lot (75%)")
	page.AssertHtmlQuery("#trust-form", "completely (90%)")
	if got := page.Attr("#trust-form", "action"); got != "/web/profile/trust" {
		t.Errorf("trust form action = %q, want /web/profile/trust", got)
	}
	if got := page.Attr("input[name=entity]", "value"); got != testEntityUri().Unadorned() {
		t.Errorf("entity hidden field = %q, want %q", got, testEntityUri().Unadorned())
	}
	if page.Find("#trust-status") != "" {
		t.Error("expected no already-trusted status when the user has not trusted this entity")
	}
}

func TestEntityPageTrustHiddenWhenLoggedOut(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	wt.AuthCookie = nil
	page := wt.GetPage(testEntityPath())
	page.AssertSuccessResponse()
	if page.Find("#trust-form") != "" {
		t.Error("expected no trust form when logged out")
	}
	if page.Find("#trust-status") != "" {
		t.Error("expected no trust status when logged out")
	}
}

func TestEntityPageShowsTrustLabelWhenAlreadyTrusted(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	stored, err := datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	stored.AddTrustRoot(testEntityUri(), 0.50)
	datastore.ActiveDataStore.StoreUser(context.TODO(), stored)

	page := wt.GetPage(testEntityPath())
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#trust-status", "You trust this entity somewhat (50%)")
	if page.Find("#trust-form") != "" {
		t.Error("expected no trust form when the entity is already trusted")
	}
}

func TestAddTrustRequiresLogin(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	wt.AuthCookie = nil
	page := wt.PostFormData("/web/profile/trust", url.Values{
		"entity": {testEntityUri().Unadorned()},
		"level":  {"0.50"},
	})
	page.AssertErrorResponse()
	page.AssertHtmlQuery("#message", "Not logged in")
}

func TestAddTrustFromEntityPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.PostFormData("/web/profile/trust", url.Values{
		"entity": {testEntityUri().Unadorned()},
		"level":  {"0.75"},
	})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#common_name", "Mr Tester")
	page.AssertHtmlQuery("#trust-status", "You trust this entity a lot (75%)")
	if page.Find("#trust-form") != "" {
		t.Error("expected trust form to be replaced by the stored level after add")
	}

	stored, err := datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	level, ok := stored.TrustLevelFor(testEntityUri())
	if !ok || level != 0.75 {
		t.Errorf("stored trust = (%v, %v), want (0.75, true)", level, ok)
	}
}

func TestAddTrustDoesNotReplaceExistingLevel(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	stored, err := datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	stored.AddTrustRoot(testEntityUri(), 0.20)
	datastore.ActiveDataStore.StoreUser(context.TODO(), stored)

	page := wt.PostFormData("/web/profile/trust", url.Values{
		"entity": {testEntityUri().Unadorned()},
		"level":  {"0.90"},
	})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#trust-status", "You trust this entity slightly (20%)")

	stored, err = datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.TrustRoots) != 1 {
		t.Fatalf("TrustRoots len = %d, want 1", len(stored.TrustRoots))
	}
	if stored.TrustRoots[0].TrustLevel != 0.20 {
		t.Errorf("TrustLevel = %v, want 0.20", stored.TrustRoots[0].TrustLevel)
	}
}

func TestAddTrustIgnoresInvalidLevel(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.PostFormData("/web/profile/trust", url.Values{
		"entity": {testEntityUri().Unadorned()},
		"level":  {"1.5"},
	})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#trust-form", "I trust this entity")
	if page.Find("#trust-status") != "" {
		t.Error("invalid level should leave the entity untrusted")
	}

	stored, err := datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.TrustRoots) != 0 {
		t.Errorf("expected no trust roots after invalid level, got %v", stored.TrustRoots)
	}
}

func TestAddTrustIgnoresUnknownEntity(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	unknown := "hash://sha256/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	page := wt.PostFormData("/web/profile/trust", url.Values{
		"entity": {unknown},
		"level":  {"0.50"},
	})
	page.AssertErrorResponse()

	stored, err := datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.TrustRoots) != 0 {
		t.Errorf("expected no trust roots for unknown entity, got %v", stored.TrustRoots)
	}
}

func TestParsePostedTrustLevel(t *testing.T) {
	cases := []struct {
		value string
		level float64
		ok    bool
	}{
		{"0", 0, true},
		{"0.21", 0.21, true},
		{"1", 1, true},
		{"-0.1", 0, false},
		{"1.5", 0, false},
		{"", 0, false},
		{"nope", 0, false},
	}
	for _, tc := range cases {
		level, ok := parsePostedTrustLevel(tc.value)
		if ok != tc.ok || level != tc.level {
			t.Errorf("parsePostedTrustLevel(%q) = (%v, %v), want (%v, %v)", tc.value, level, ok, tc.level, tc.ok)
		}
	}
}

func TestTrustLevelLabel(t *testing.T) {
	cases := []struct {
		level float64
		want  string
	}{
		{0, "slightly (20%)"},
		{0.20, "slightly (20%)"},
		{0.21, "slightly (20%)"},
		{0.50, "somewhat (50%)"},
		{0.60, "somewhat (50%)"},
		{0.75, "a lot (75%)"},
		{0.80, "a lot (75%)"},
		{0.90, "completely (90%)"},
		{1, "completely (90%)"},
	}
	for _, tc := range cases {
		if got := trustLevelLabel(tc.level); got != tc.want {
			t.Errorf("trustLevelLabel(%v) = %q, want %q", tc.level, got, tc.want)
		}
	}
}
