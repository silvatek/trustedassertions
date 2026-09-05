package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
	"silvatek.uk/trustedassertions/internal/webtest"
)

func TestProfileShowsPasskeyControls(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.GetPage("/web/profile")
	page.AssertHtmlQuery("h3", "Passkeys")
	page.AssertHtmlQuery("#add-passkey", "Add passkey")
	page.AssertHtmlQuery("#no-passkeys", "No passkeys registered.")
	if page.Find(".revoke-passkey") != "" {
		t.Error("expected no revoke link when there are no passkeys")
	}
	if page.Attr("script[src*='passkey.js']", "src") != "/web/static/passkey.js" {
		t.Errorf("passkey script src = %q", page.Attr("script[src*='passkey.js']", "src"))
	}
}

func TestProfileListsPasskeys(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	pk := auth.Passkey{ID: []byte("cred-1"), Name: "Laptop", PublicKey: []byte{1}}
	if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
		t.Fatal(err)
	}

	page := wt.GetPage("/web/profile")
	page.AssertHtmlQuery(".passkey-name", "Laptop")
	page.AssertHtmlQuery(".passkey-last-used", "Never")
	page.AssertHtmlQuery(".revoke-passkey", "Revoke")
	if page.Attr(".revoke-passkey", "hx-boost") != "false" {
		t.Errorf("revoke hx-boost = %q", page.Attr(".revoke-passkey", "hx-boost"))
	}
	wantID := base64.RawURLEncoding.EncodeToString([]byte("cred-1"))
	if page.Attr(".revoke-passkey", "data-id") != wantID {
		t.Errorf("data-id = %q, want %q", page.Attr(".revoke-passkey", "data-id"), wantID)
	}
}

func TestPasskeyRegisterBeginRequiresAuth(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	page := wt.PostJSON("/web/passkey/register/begin", nil)
	if page.Status() != 401 {
		t.Errorf("status = %d, want 401", page.Status())
	}
}

func TestPasskeyRegisterBeginReturnsOptions(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.PostJSON("/web/passkey/register/begin", nil)
	if page.Status() != 200 {
		t.Fatalf("status = %d, body = %s", page.Status(), page.RawBody())
	}

	var body map[string]any
	if err := json.Unmarshal(page.RawBody(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	publicKey, ok := body["publicKey"].(map[string]any)
	if !ok {
		t.Fatalf("publicKey missing: %s", page.RawBody())
	}
	if publicKey["challenge"] == "" {
		t.Error("expected challenge")
	}
	selection, _ := publicKey["authenticatorSelection"].(map[string]any)
	if selection["userVerification"] != "required" {
		t.Errorf("userVerification = %v", selection["userVerification"])
	}
	page.AssertHasCookie("webauthn_session")
}

func TestPasskeyRegisterBeginRejectsCap(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	for i := 0; i < auth.MaxPasskeys; i++ {
		pk := auth.Passkey{ID: []byte{byte(i + 1)}, PublicKey: []byte{1}}
		if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
			t.Fatal(err)
		}
	}

	page := wt.PostJSON("/web/passkey/register/begin", nil)
	if page.Status() != 409 {
		t.Errorf("status = %d, want 409", page.Status())
	}
}

func TestPasskeyRegisterFinishRequiresSession(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	page := wt.PostJSON("/web/passkey/register/finish", []byte(`{}`))
	if page.Status() != 400 {
		t.Errorf("status = %d, want 400", page.Status())
	}
}

func TestLoginPageShowsPasskeyControls(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	page := wt.GetPage("/web/login")
	page.AssertHtmlQuery("#use-passkey", "Use a passkey")
	if page.Attr("#use-passkey", "hx-boost") != "false" {
		t.Errorf("use-passkey hx-boost = %q", page.Attr("#use-passkey", "hx-boost"))
	}
	if page.Attr("script[src*='passkey.js']", "src") != "/web/static/passkey.js" {
		t.Errorf("passkey script src = %q", page.Attr("script[src*='passkey.js']", "src"))
	}
}

func TestPasskeyLoginBeginUnknownUser(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	page := wt.PostJSON("/web/passkey/login/begin", []byte(`{"user_id":"nobody"}`))
	assertPasskeyLoginFailure(t, page)
	assertNoAuthCookie(t, wt)
}

func TestPasskeyLoginBeginEmptyUserID(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	page := wt.PostJSON("/web/passkey/login/begin", []byte(`{"user_id":""}`))
	assertPasskeyLoginFailure(t, page)
	assertNoAuthCookie(t, wt)
}

func TestPasskeyLoginBeginLockedUser(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	pk := auth.Passkey{ID: []byte("cred-locked-1"), PublicKey: []byte{1, 2, 3}}
	if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
		t.Fatal(err)
	}
	stored, err := datastore.ActiveDataStore.FetchUser(context.TODO(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	stored.SetStatus(auth.UserStatusLocked)
	datastore.ActiveDataStore.StoreUser(context.TODO(), stored)

	page := wt.PostJSON("/web/passkey/login/begin", []byte(`{"user_id":"`+user.Id+`"}`))
	assertPasskeyLoginFailure(t, page)
	assertNoAuthCookie(t, wt)
}

func TestPasskeyLoginBeginUserWithoutPasskeys(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	page := wt.PostJSON("/web/passkey/login/begin", []byte(`{"user_id":"`+user.Id+`"}`))
	assertPasskeyLoginFailure(t, page)
	assertNoAuthCookie(t, wt)
}

func TestPasskeyLoginBeginReturnsOptions(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	pk := auth.Passkey{ID: []byte("cred-login-1"), PublicKey: []byte{1, 2, 3}}
	if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
		t.Fatal(err)
	}

	page := wt.PostJSON("/web/passkey/login/begin", []byte(`{"user_id":"`+user.Id+`"}`))
	if page.Status() != 200 {
		t.Fatalf("status = %d, body = %s", page.Status(), page.RawBody())
	}

	var body map[string]any
	if err := json.Unmarshal(page.RawBody(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	publicKey, ok := body["publicKey"].(map[string]any)
	if !ok {
		t.Fatalf("publicKey missing: %s", page.RawBody())
	}
	if publicKey["challenge"] == "" {
		t.Error("expected challenge")
	}
	if publicKey["userVerification"] != "required" {
		t.Errorf("userVerification = %v", publicKey["userVerification"])
	}
	allow, ok := publicKey["allowCredentials"].([]any)
	if !ok || len(allow) == 0 {
		t.Errorf("allowCredentials = %v", publicKey["allowCredentials"])
	}
	page.AssertHasCookie("webauthn_session")
	assertNoAuthCookie(t, wt)
}

func TestPasskeyLoginFinishRequiresSession(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	page := wt.PostJSON("/web/passkey/login/finish", []byte(`{}`))
	assertPasskeyLoginFailure(t, page)
	assertNoAuthCookie(t, wt)
}

func TestPasskeyRevokeRequiresAuth(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	id := base64.RawURLEncoding.EncodeToString([]byte("cred-1"))
	page := wt.PostJSON("/web/passkey/revoke", []byte(`{"id":"`+id+`"}`))
	if page.Status() != 401 {
		t.Errorf("status = %d, want 401", page.Status())
	}
}

func TestPasskeyRevokeSuccess(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	pk := auth.Passkey{ID: []byte("cred-revoke-1"), Name: "Laptop", PublicKey: []byte{1}}
	if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
		t.Fatal(err)
	}

	id := base64.RawURLEncoding.EncodeToString(pk.ID)
	page := wt.PostJSON("/web/passkey/revoke", []byte(`{"id":"`+id+`"}`))
	if page.Status() != 200 {
		t.Fatalf("status = %d, body = %s", page.Status(), page.RawBody())
	}

	var body map[string]string
	if err := json.Unmarshal(page.RawBody(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
	assertAuthCookieUnchanged(t, page)

	stored, err := datastore.ActiveDataStore.FetchUser(context.TODO(), user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Passkeys) != 0 {
		t.Errorf("passkeys remaining = %d, want 0", len(stored.Passkeys))
	}

	profile := wt.GetPage("/web/profile")
	profile.AssertHtmlQuery("#no-passkeys", "No passkeys registered.")
}

func TestPasskeyRevokeEmptyID(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	pk := auth.Passkey{ID: []byte("cred-keep-1"), PublicKey: []byte{1}}
	if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
		t.Fatal(err)
	}

	page := wt.PostJSON("/web/passkey/revoke", []byte(`{"id":""}`))
	if page.Status() != 400 {
		t.Errorf("status = %d, want 400, body = %s", page.Status(), page.RawBody())
	}
	assertPasskeyStillStored(t, user.Id, pk.ID)
}

func TestPasskeyRevokeInvalidID(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	pk := auth.Passkey{ID: []byte("cred-keep-2"), PublicKey: []byte{1}}
	if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
		t.Fatal(err)
	}

	page := wt.PostJSON("/web/passkey/revoke", []byte(`{"id":"%%%"}`))
	if page.Status() != 400 {
		t.Errorf("status = %d, want 400, body = %s", page.Status(), page.RawBody())
	}
	assertPasskeyStillStored(t, user.Id, pk.ID)
}

func TestPasskeyRevokeUnknownID(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	pk := auth.Passkey{ID: []byte("cred-keep-3"), PublicKey: []byte{1}}
	if err := datastore.AddPasskey(context.TODO(), user.Id, pk); err != nil {
		t.Fatal(err)
	}

	id := base64.RawURLEncoding.EncodeToString([]byte("missing-cred"))
	page := wt.PostJSON("/web/passkey/revoke", []byte(`{"id":"`+id+`"}`))
	if page.Status() != 404 {
		t.Errorf("status = %d, want 404, body = %s", page.Status(), page.RawBody())
	}
	assertPasskeyStillStored(t, user.Id, pk.ID)
}

func TestPasskeyLoginFinishRejectsGarbageBody(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()
	wt.AuthCookie = nil

	page := wt.PostJSON("/web/passkey/login/finish", []byte(`not-json`))
	assertPasskeyLoginFailure(t, page)
	assertNoAuthCookie(t, wt)
}

func assertPasskeyLoginFailure(t *testing.T, page *webtest.WebPage) {
	t.Helper()
	if page.Status() != 401 {
		t.Errorf("status = %d, want 401, body = %s", page.Status(), page.RawBody())
	}
	var body map[string]string
	if err := json.Unmarshal(page.RawBody(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["error"] != passkeyLoginFailMessage {
		t.Errorf("error = %q, want %q", body["error"], passkeyLoginFailMessage)
	}
}

func assertAuthCookieUnchanged(t *testing.T, page *webtest.WebPage) {
	t.Helper()
	setCookie := page.Header("Set-Cookie")
	if setCookie == "" {
		return
	}
	if strings.Contains(setCookie, "auth=") {
		t.Errorf("auth cookie should not be set or cleared, Set-Cookie = %q", setCookie)
	}
}

func assertPasskeyStillStored(t *testing.T, userID string, credentialID []byte) {
	t.Helper()
	stored, err := datastore.ActiveDataStore.FetchUser(context.TODO(), userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, pk := range stored.Passkeys {
		if string(pk.ID) == string(credentialID) {
			return
		}
	}
	t.Errorf("expected passkey %q to remain stored", credentialID)
}

func assertNoAuthCookie(t *testing.T, wt *webtest.WebTest) {
	t.Helper()
	origin, _ := url.Parse(wt.Server.URL + "/")
	for _, cookie := range wt.Client.Jar.Cookies(origin) {
		if cookie.Name == "auth" && cookie.Value != "" {
			t.Error("auth cookie should not be set")
		}
	}
}
