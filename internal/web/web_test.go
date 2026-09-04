package web

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/assertions"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/entities"
	. "silvatek.uk/trustedassertions/internal/references"
	"silvatek.uk/trustedassertions/internal/testdata"
	"silvatek.uk/trustedassertions/internal/webtest"
)

var user *auth.User

func NewWebTest(t *testing.T) *webtest.WebTest {
	TemplateDir = "../../web"

	datastore.InitInMemoryDataStore()
	assertions.PublicKeyResolver = datastore.ActiveDataStore

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer := entities.NewEntity("Signing entity", *big.NewInt(123456))
	signer.MakeCertificate(privateKey)
	datastore.ActiveDataStore.Store(context.Background(), &signer)
	datastore.ActiveDataStore.StoreKey(signer.Uri(), entities.PrivateKeyToString(privateKey))
	DefaultEntityUri = signer.Uri()

	testdata.SetupTestData(context.Background(), "../../testdata", signer.Uri().String(), entities.PrivateKeyToString(privateKey))

	wt := webtest.MakeWebTest(t)

	router := mux.NewRouter()
	AddHandlers(router)

	user = &auth.User{Id: "admin"}
	wt.Passwd = "testing"
	user.HashPassword(wt.Passwd)
	user.KeyRefs = append(user.KeyRefs, auth.KeyRef{UserId: user.Id, KeyId: signer.Uri().Unadorned(), Summary: ""})
	user.AddRole(auth.RoleAuthor)
	user.AddRole(auth.RoleAdministrator)
	wt.AuthCookie = MakeAuthCookie(user.Id)
	datastore.ActiveDataStore.StoreUser(context.TODO(), *user)

	if err := InitWebAuthn("http://127.0.0.1:8080", "Trusted Assertions", "default_csrf_key"); err != nil {
		t.Fatalf("InitWebAuthn: %v", err)
	}

	wt.Server = httptest.NewServer(router)

	return wt
}

func TestHomePage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertHtmlQuery("h1", "Trusted Assertions")
}

func TestErrorPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/broken")
	page.AssertHtmlQuery("#intro", "Sorry, an error has occurred.")
	page.AssertHtmlQuery("#message", "Fake error for testing")
}

func TestStatementPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.AssertHtmlQuery("#content", "The universe exists")
}

func TestEntityPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/entities/177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db")
	page.AssertHtmlQuery("#common_name", "Mr Tester")
}

func TestAssertionPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/assertions/514518bb09d57524bc6b96842721e4c4404cb4a3329aadf1761bb3eddb2832da")
	page.AssertHtmlQuery("#category", "IsTrue")
}

func TestNewStatementPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/newstatement")
	page.AssertHtmlQuery("h2", "New Statement")
}

func TestPostNewStatement(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	data := url.Values{
		"statement": {"Test statement"},
		"sign_as":   {user.KeyRefs[0].KeyId},
	}
	page := wt.PostFormData("/web/newstatement", data)
	page.AssertSuccessResponse()

	newUri := UriFromString(strings.TrimSpace(page.Find("#uri")))

	// Make sure the new assertion is really in the datastore
	_, err := datastore.ActiveDataStore.FetchAssertion(context.TODO(), newUri)
	if err != nil {
		t.Errorf("Error fetching new assertion: %v", err)
	}
}

func TestNewEntity(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/newentity")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("label", "Entity name")

	page = wt.PostFormData("/web/newentity", url.Values{"commonname": {"Test entity"}})
	page.AssertSuccessResponse()
	uri := UriFromString(page.Find("span.fulluri"))

	newEntity, err := datastore.ActiveDataStore.FetchEntity(context.TODO(), uri)
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

	page := wt.GetPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.AssertHtmlQuery("a", "Add a new assertion for this statement.")

	page = wt.GetPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f/addassertion")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("label", "Assertion:")

	values := url.Values{
		"assertion_type": {"IsTrue"},
		"confidence":     {"0.75"},
		"sign_as":        {user.KeyRefs[0].KeyId},
	}
	page = wt.PostFormData("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f/addassertion", values)
	page.AssertSuccessResponse()

	uri := UriFromString(page.Find("span.fulluri"))
	_, err := datastore.ActiveDataStore.FetchAssertion(context.TODO(), uri)
	if err != nil {
		t.Errorf("Error fetching new assertion")
	}

}

func TestSearch(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertHtmlQuery("#searchform", "Search for")
	if page.Attr("#searchform", "hx-target") != "#searchresults" {
		t.Errorf("Expected search form to target #searchresults, got %s", page.Attr("#searchform", "hx-target"))
	}

	page = wt.GetPage("/web/search?query=universe")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#searchform", "Search for")
	page.AssertHtmlQuery("h2", "Search results")
	page.AssertHtmlQuery(".searchresults", "The universe exists")
}

func TestSearchQueryEscaped(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	payload := `" onfocus="alert(1)" x="`
	page := wt.GetPage("/web/search?query=" + url.QueryEscape(payload))
	page.AssertSuccessResponse()
	page.AssertHtmlQueryEscaped("#searchterm", payload)
}

func TestSearchHtmx(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPageWithHeaders("/web/search?query=universe", map[string]string{"HX-Request": "true"})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#searchform", "Search for")
	page.AssertHtmlQuery("#searchresults", "The universe exists")
	page.AssertHtmlQuery(".searchresults", "The universe exists")
	page.AssertHtmxFragment(true)
}

func TestQrCode(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.AssertHtmlQuery("#content", "The universe exists")

	page = wt.GetPage("/web/share?hash=33fe9d5eedb329c5a662d3c206d8938a33f94795c3f715be0bcd53fbdcadc7e8&type=entity")
	page.AssertHtmlQuery("h2", "Share Item")
	page.AssertSuccessResponse()
}

func TestHtmxFullPage(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/home")
	page.AssertSuccessResponse()
	page.AssertHtmxFragment(false)

	page = wt.GetPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.AssertSuccessResponse()
	page.AssertHtmxFragment(false)
}

func TestHtmxFragment(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPageWithHeaders("/web/home", map[string]string{"HX-Request": "true"})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#searchform", "Search for")
	page.AssertHtmlQuery("#page", "Search for")
	page.AssertHtmxFragment(true)
}
