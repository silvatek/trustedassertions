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

func setup(t *testing.T) *webtest.WebTest {
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
	wt.AuthCookie = MakeAuthCookie(user.Id)
	datastore.ActiveDataStore.StoreUser(context.TODO(), *user)

	wt.Server = httptest.NewServer(router)

	return wt
}

func TestHomePage(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertHtmlQuery("h1", "Trusted Assertions")
}

func TestErrorPage(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/web/broken")
	page.AssertHtmlQuery("#intro", "Sorry, an error has occurred.")
	page.AssertHtmlQuery("#message", "Fake error for testing")
}

func TestStatementPage(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.AssertHtmlQuery("#content", "The universe exists")
}

func TestEntityPage(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/web/entities/177ed36580cf1ed395e1d0d3a7709993ac1599ee844dc4cf5b9573a1265df2db")
	page.AssertHtmlQuery("#common_name", "Mr Tester")
}

func TestAssertionPage(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/web/assertions/514518bb09d57524bc6b96842721e4c4404cb4a3329aadf1761bb3eddb2832da")
	page.AssertHtmlQuery("#category", "IsTrue")
}

func TestNewStatementPage(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/web/newstatement")
	page.AssertHtmlQuery("h2", "New Statement")
}

func TestPostNewStatement(t *testing.T) {
	wt := setup(t)
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
	wt := setup(t)
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
	wt := setup(t)
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
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/")
	page.AssertHtmlQuery("#searchform", "Search for")

	page = wt.PostFormData("/web/search", url.Values{"query": {"universe"}})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h2", "Search results")
	page.AssertHtmlQuery("#content", "The universe exists")
}

func TestLoginLogout(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	wt.AuthCookie = nil

	page := wt.GetPage("/web/login")
	page.AssertNoCookie("auth")
	page.AssertHtmlQuery("h2", "Login")

	page = wt.PostFormData("/web/login", url.Values{"user_id": {user.Id}, "password": {wt.Passwd}})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("span", user.Id)
	page.AssertHasCookie("auth")

	page = wt.GetPage("/web/logout")
	page.AssertHtmlQuery("#message", "You have been logged out")
	page.AssertNoCookie("auth")
}

func TestBadLogin(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.PostFormData("/web/login", url.Values{"user_id": {"jkdshffkdjshdskfjhd"}, "password": {wt.Passwd}})
	page.AssertHtmlQuery(".error", "Unable to verify identity")

	page = wt.PostFormData("/web/login", url.Values{"user_id": {user.Id}, "password": {"jkdfhskjfdshfk"}})
	page.AssertHtmlQuery(".error", "Unable to verify identity")
}

func TestQrCode(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	page := wt.GetPage("/web/statements/e88688ef18e5c82bb8ea474eceeac8c6eb81d20ec8d903750753d3137865d10f")
	page.AssertHtmlQuery("#content", "The universe exists")

	page = wt.GetPage("/web/share?hash=33fe9d5eedb329c5a662d3c206d8938a33f94795c3f715be0bcd53fbdcadc7e8&type=entity")
	page.AssertHtmlQuery("h2", "Share Item")
	page.AssertSuccessResponse()
}

func TestRegistration(t *testing.T) {
	wt := setup(t)
	defer wt.Close()

	ctx := context.Background()

	datastore.ActiveDataStore.StoreRegistration(ctx, auth.Registration{Code: "ABC", Status: "Pending"})

	page := wt.GetPage("/web/register")
	page.AssertHtmlQuery("h2", "User Registration")

	page = wt.PostFormData("/web/register", url.Values{"reg_code": {"ABC"}, "user_id": {"Tester 99"}, "password1": {"jsdj87sda;swg59jmd;;874j"}, "password2": {"jsdj87sda;swg59jmd;;874j"}})
	page.AssertHtmlQuery("h2", "Login")
	page.AssertHtmlQuery(".error", "")
}
