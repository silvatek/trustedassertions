package main

import (
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
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
	page.assertHtmlQuery("h1", "New Statement")
}

func TestPostNewStatement(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	data := url.Values{
		"statement": {"Test statement"},
		"sign_as":   {wt.user.KeyRefs[0].KeyId},
	}
	response, err := postFormData(wt.server.URL+"/web/newstatement", data, wt.user)
	if err != nil {
		t.Errorf("Error posting to %s : %v", "/web/newstatement", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		t.Errorf("Error posting to %s : %d", "/web/newstatement", response.StatusCode)
		return
	}

	html, _ := goquery.NewDocumentFromReader(response.Body)

	newUri := assertions.UriFromString(strings.TrimSpace(html.Find("#uri").Text()))

	// Make sure the new assertion is really in the datastore
	_, err = datastore.ActiveDataStore.FetchAssertion(newUri)
	if err != nil {
		t.Errorf("Error fetching new assertion: %v", err)
	}
}

func TestLoginLogout(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.getPage("/web/login")

	page.assertHtmlQuery("h2", "Login")

	page = wt.getPage("/web/logout")
	page.assertHtmlQuery("#message", "You have been logged out")
}
