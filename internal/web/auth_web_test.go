package web

import (
	"context"
	"net/url"
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestLoginLogout(t *testing.T) {
	wt := NewWebTest(t)
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
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.PostFormData("/web/login", url.Values{"user_id": {"jkdshffkdjshdskfjhd"}, "password": {wt.Passwd}})
	page.AssertHtmlQuery(".error", "Unable to verify identity")

	page = wt.PostFormData("/web/login", url.Values{"user_id": {user.Id}, "password": {"jkdfhskjfdshfk"}})
	page.AssertHtmlQuery(".error", "Unable to verify identity")
}

func TestProfileShowsRoles(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/profile")
	page.AssertHtmlQuery("h3", "Roles")
	page.AssertHtmlQuery(".user-role", auth.RoleAuthor)
	page.AssertHtmlQuery(".user-role", auth.RoleAdministrator)
	if page.Find("#no-roles") != "" {
		t.Error("expected no empty-roles message when roles are assigned")
	}
}

func TestProfileShowsNoRoles(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	stored, err := datastore.ActiveDataStore.FetchUser(context.Background(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	stored.Roles = nil
	datastore.ActiveDataStore.StoreUser(context.TODO(), stored)

	page := wt.GetPage("/web/profile")
	page.AssertHtmlQuery("#no-roles", "No roles assigned.")
	if page.Find(".user-role") != "" {
		t.Error("expected no role items when none are assigned")
	}
}

func TestRegistration(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	ctx := context.Background()

	datastore.ActiveDataStore.StoreRegistration(ctx, auth.Registration{Code: "abc", Status: "Pending"})

	page := wt.GetPage("/web/register")
	page.AssertHtmlQuery("h2", "User Registration")

	page = wt.PostFormData("/web/register", url.Values{"reg_code": {"ABC"}, "user_id": {"Tester 99"}, "password1": {"jsdj87sda;swg59jmd;;874j"}, "password2": {"jsdj87sda;swg59jmd;;874j"}})
	page.AssertHtmlQuery("h2", "Login")
	page.AssertHtmlQuery(".error", "")
}
