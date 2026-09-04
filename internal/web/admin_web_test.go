package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestAdminPageRequiresLogin(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	wt.AuthCookie = nil
	page := wt.GetPage("/web/admin")
	page.AssertErrorResponse()
	page.AssertHtmlQuery("#message", "Not logged in")
}

func TestAdminPageRequiresAdministrator(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	author := auth.User{Id: "authoronly"}
	author.HashPassword("testing")
	author.AddRole(auth.RoleAuthor)
	datastore.ActiveDataStore.StoreUser(context.TODO(), author)
	wt.AuthCookie = MakeAuthCookie(author.Id)

	page := wt.GetPage("/web/admin")
	page.AssertErrorResponse()
	page.AssertHtmlQuery("#message", "Not authorized")

	page = wt.PostFormData("/web/admin/invites", url.Values{"role": {auth.RoleAuthor}})
	page.AssertErrorResponse()
}

func TestAdminMenuLinkForAdministrator(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/home")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery(`a[href="/web/admin"]`, "Admin")
}

func TestAdminMenuLinkHiddenForAuthor(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	author := auth.User{Id: "authoronly"}
	author.HashPassword("testing")
	author.AddRole(auth.RoleAuthor)
	datastore.ActiveDataStore.StoreUser(context.TODO(), author)
	wt.AuthCookie = MakeAuthCookie(author.Id)

	page := wt.GetPage("/web/home")
	page.AssertSuccessResponse()
	if page.Attr(`a[href="/web/admin"]`, "href") != "" {
		t.Error("non-admin should not see the Admin menu link")
	}
}

func TestAdminMenuLinkHiddenWhenLoggedOut(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	wt.AuthCookie = nil
	page := wt.GetPage("/web/home")
	page.AssertSuccessResponse()
	if page.Attr(`a[href="/web/admin"]`, "href") != "" {
		t.Error("anonymous visitors should not see the Admin menu link")
	}
}

func TestAdminCreatesAndListsInvite(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/admin")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h2", "Admin")
	page.AssertHtmlQuery("h3", "Create registration code")
	page.AssertHtmlQuery("h3", "Registrations")
	if page.Attr("#role-author", "checked") != "checked" {
		t.Error("Author should be checked by default")
	}
	if page.Attr("#role-administrator", "checked") != "" {
		t.Error("Administrator should be unchecked by default")
	}

	page = wt.PostFormData("/web/admin/invites", url.Values{"role": {auth.RoleAuthor}})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("#created-invite", "New registration code:")
	code := strings.TrimSpace(page.Find("#created-invite .invite-code"))
	words := strings.Fields(code)
	if len(words) != auth.InviteCodeWordCount {
		t.Fatalf("created invite %q should have %d words", code, auth.InviteCodeWordCount)
	}
	if code != strings.ToLower(code) {
		t.Errorf("stored invite should be lowercase: %q", code)
	}
	page.AssertHtmlQuery(".registration .invite-code", code)
	page.AssertHtmlQuery(".registration .invite-roles", auth.RoleAuthor)
	page.AssertHtmlQuery(".registration .invite-status", "Pending")
	page.AssertHtmlQuery(".registration .invite-created-by", user.Id)
	today := time.Now().UTC().Format("2006-01-02")
	page.AssertHtmlQuery(".registration .invite-created", today)
	page.AssertHtmlQuery(".registration .invite-expires", time.Now().UTC().Add(auth.InviteValidity).Format("2006-01-02"))
	page.AssertHtmlQuery(".registration .invite-completed", "Never")

	complete := auth.Registration{
		Code:   "used tree blue sky",
		Status: "Complete",
		Roles:  []string{auth.RoleAuthor},
	}
	datastore.ActiveDataStore.StoreRegistration(context.Background(), complete)
	page = wt.GetPage("/web/admin")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery(".registration .invite-code", complete.Code)
	page.AssertHtmlQuery(".registration .invite-status", "Complete")

	reg, err := datastore.ActiveDataStore.FetchRegistration(context.Background(), code)
	if err != nil {
		t.Fatalf("stored registration not found: %v", err)
	}
	if reg.Status != "Pending" {
		t.Errorf("expected Pending status, got %s", reg.Status)
	}
	if len(reg.Roles) != 1 || reg.Roles[0] != auth.RoleAuthor {
		t.Errorf("expected Author role on invite, got %v", reg.Roles)
	}
	if reg.CreatedBy != user.Id {
		t.Errorf("CreatedBy = %q, want %s", reg.CreatedBy, user.Id)
	}
	if reg.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set when the invite is created")
	}
	if !reg.ExpiresAt.Equal(reg.CreatedAt.Add(auth.InviteValidity)) {
		t.Errorf("ExpiresAt = %v, want CreatedAt + %s", reg.ExpiresAt, auth.InviteValidity)
	}
}

func TestAdminJwtContainsOnlyUserId(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/admin")
	page.AssertSuccessResponse()

	parts := strings.Split(wt.AuthCookie.Value, ".")
	if len(parts) < 2 {
		t.Fatalf("auth cookie is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("jwt payload: %v", err)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("jwt json: %v", err)
	}
	if _, ok := claims["roles"]; ok {
		t.Errorf("JWT should not include roles, got %s", payload)
	}
	if strings.Contains(string(payload), auth.RoleAdministrator) || strings.Contains(string(payload), auth.RoleAuthor) {
		t.Errorf("JWT should not include role labels, got %s", payload)
	}
	if claims["sub"] != user.Id {
		t.Errorf("JWT sub = %v, want %s", claims["sub"], user.Id)
	}
}
