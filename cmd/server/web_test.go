package main

import (
	"net/url"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestHomePage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/")
	page.assertHtmlQuery("h1", "Trusted Assertions")
}

func TestErrorPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/broken")
	page.assertHtmlQuery("#intro", "Sorry, an error has occurred.")
	page.assertHtmlQuery("#message", "Fake error for testing")
}

func TestStatementPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.assertHtmlQuery("#content", "The universe exists")
}

func TestEntityPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/entities/177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db")
	page.assertHtmlQuery("#common_name", "Mr Tester")
}

func TestAssertionPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/assertions/514518bb09d57524bc6b96842721e4c4404cb4a3329aadf1761bb3eddb2832da")
	page.assertHtmlQuery("#category", "IsTrue")
}

func TestNewStatementPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/newstatement")
	page.assertHtmlQuery("h2", "New Statement")
}

func TestPostNewStatement(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	data := url.Values{
		"statement": {"Test statement"},
		"sign_as":   {wt.user.KeyRefs[0].KeyId},
	}
	page := wt.postFormData("/web/newstatement", data)
	page.assertSuccessResponse()

	newUri := assertions.UriFromString(strings.TrimSpace(page.Find("#uri")))

	// Make sure the new assertion is really in the datastore
	_, err := datastore.ActiveDataStore.FetchAssertion(newUri)
	if err != nil {
		t.Errorf("Error fetching new assertion: %v", err)
	}
}

func TestNewEntity(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/newentity")
	page.assertSuccessResponse()
	page.assertHtmlQuery("label", "Entity name")

	page = wt.postFormData("/web/newentity", url.Values{"commonname": {"Test entity"}})
	page.assertSuccessResponse()
	uri := assertions.UriFromString(page.Find("span.fulluri"))

	newEntity, err := datastore.ActiveDataStore.FetchEntity(uri)
	if err != nil {
		t.Errorf("Unable to fetch new entity: %v", err)
	}
	if newEntity.CommonName != "Test entity" {
		t.Errorf("Could not find new entity with correct name")
	}
}

func TestAddAssertion(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.assertHtmlQuery("a", "Add a new assertion for this statement.")

	page = wt.getPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f/addassertion")
	page.assertSuccessResponse()
	page.assertHtmlQuery("label", "Assertion:")

	values := url.Values{
		"assertion_type": {"IsTrue"},
		"confidence":     {"0.75"},
		"sign_as":        {wt.user.KeyRefs[0].KeyId},
	}
	page = wt.postFormData("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f/addassertion", values)
	page.assertSuccessResponse()

	uri := assertions.UriFromString(page.Find("span.fulluri"))
	_, err := datastore.ActiveDataStore.FetchAssertion(uri)
	if err != nil {
		t.Errorf("Error fetching new assertion")
	}

}

func TestSearch(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/")
	page.assertHtmlQuery("#searchform", "Search for")

	page = wt.postFormData("/web/search", url.Values{"query": {"universe"}})
	page.assertSuccessResponse()
	page.assertHtmlQuery("h2", "Search results")
	page.assertHtmlQuery("#content", "The universe exists")
}

func TestLoginLogout(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/login")
	page.assertNoCookie("auth")
	page.assertHtmlQuery("h2", "Login")

	page = wt.postFormData("/web/login", url.Values{"user_id": {wt.user.Id}, "password": {wt.passwd}})
	page.assertSuccessResponse()
	page.assertHtmlQuery("#username", wt.user.Id)
	page.assertHasCookie("auth")

	page = wt.getPage("/web/logout")
	page.assertHtmlQuery("#message", "You have been logged out")
	page.assertNoCookie("auth")
}

func TestBadLogin(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.postFormData("/web/login", url.Values{"user_id": {"jkdshffkdjshdskfjhd"}, "password": {wt.passwd}})
	page.assertHtmlQuery(".error", "Unable to verify identity")

	page = wt.postFormData("/web/login", url.Values{"user_id": {wt.user.Id}, "password": {"jkdfhskjfdshfk"}})
	page.assertHtmlQuery(".error", "Unable to verify identity")
}
