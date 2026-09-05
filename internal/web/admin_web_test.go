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

	page = wt.PostFormData("/web/admin/users", url.Values{"user_id": {user.Id}, "action": {"lock"}})
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

func TestAdminListsUsers(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	alice := auth.User{Id: "alice"}
	alice.AddRole(auth.RoleAuthor)
	datastore.ActiveDataStore.StoreUser(context.TODO(), alice)
	datastore.ActiveDataStore.StoreUser(context.TODO(), auth.User{Id: "nobody"})

	page := wt.GetPage("/web/admin")
	page.AssertSuccessResponse()
	page.AssertHtmlQuery("h3", "Users")
	page.AssertHtmlQuery("#users thead", "User ID")
	page.AssertHtmlQuery("#users thead", "Roles")
	page.AssertHtmlQuery("#users thead", "Status")
	page.AssertHtmlQuery("#users thead", "Actions")
	page.AssertHtmlQuery("#users .user-id", user.Id)
	page.AssertHtmlQuery("#users .user-id", "alice")
	page.AssertHtmlQuery("#users .user-id", "nobody")
	page.AssertHtmlQuery("#users .user-roles", auth.RoleAuthor)
	page.AssertHtmlQuery("#users .user-roles", auth.RoleAdministrator)
	page.AssertHtmlQuery("#users .user-roles", "No roles")
	page.AssertHtmlQuery("#users .user-status", "Active")
	page.AssertHtmlQuery(`tr.user[data-user-id="alice"] .lock-user`, "Lock")
	page.AssertHtmlQuery(`tr.user[data-user-id="nobody"] .lock-user`, "Lock")
	if page.Attr(`tr.user[data-user-id="alice"] a.lock-user`, "href") != "#" {
		t.Error("lock control should be an action link")
	}
	if page.Attr(`tr.user[data-user-id="alice"] .lock-user`, "hx-boost") != "false" {
		t.Errorf("lock hx-boost = %q", page.Attr(`tr.user[data-user-id="alice"] .lock-user`, "hx-boost"))
	}
	if page.Find(`tr.user[data-user-id="`+user.Id+`"] .lock-user`) != "" {
		t.Error("administrator should not see a lock control for themselves")
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

func TestAdminLocksAndUnlocksUser(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	alice := auth.User{Id: "alice"}
	alice.HashPassword("testing")
	alice.AddRole(auth.RoleAuthor)
	datastore.ActiveDataStore.StoreUser(context.TODO(), alice)

	page := wt.PostFormData("/web/admin/users", url.Values{"user_id": {"alice"}, "action": {"lock"}})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery(`tr.user[data-user-id="alice"] .user-status`, "Locked")
	page.AssertHtmlQuery(`tr.user[data-user-id="alice"] .unlock-user`, "Unlock")
	if page.Find(`tr.user[data-user-id="alice"] .lock-user`) != "" {
		t.Error("locked user should not show a lock control")
	}

	stored, err := datastore.ActiveDataStore.FetchUser(context.TODO(), "alice")
	if err != nil {
		t.Fatalf("FetchUser: %v", err)
	}
	if stored.Status != auth.UserStatusLocked {
		t.Errorf("status = %q, want %q", stored.Status, auth.UserStatusLocked)
	}
	if !stored.HasRole(auth.RoleAuthor) {
		t.Error("lock should not remove roles")
	}

	page = wt.PostFormData("/web/admin/users", url.Values{"user_id": {"alice"}, "action": {"unlock"}})
	page.AssertSuccessResponse()
	page.AssertHtmlQuery(`tr.user[data-user-id="alice"] .user-status`, "Active")
	page.AssertHtmlQuery(`tr.user[data-user-id="alice"] .lock-user`, "Lock")

	stored, err = datastore.ActiveDataStore.FetchUser(context.TODO(), "alice")
	if err != nil {
		t.Fatalf("FetchUser: %v", err)
	}
	if stored.Status != auth.UserStatusActive {
		t.Errorf("status = %q, want %q", stored.Status, auth.UserStatusActive)
	}
}

func TestAdminCannotLockSelf(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.PostFormData("/web/admin/users", url.Values{"user_id": {user.Id}, "action": {"lock"}})
	page.AssertErrorResponse()

	stored, err := datastore.ActiveDataStore.FetchUser(context.TODO(), user.Id)
	if err != nil {
		t.Fatalf("FetchUser: %v", err)
	}
	if stored.IsLocked() {
		t.Error("administrator must not be able to lock themselves")
	}
}

func TestAdminLockUnknownUser(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.PostFormData("/web/admin/users", url.Values{"user_id": {"missing"}, "action": {"lock"}})
	page.AssertErrorResponse()
}
