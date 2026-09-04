package web

import (
	"context"
	"encoding/json"
	"net/url"
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

func assertNoAuthCookie(t *testing.T, wt *webtest.WebTest) {
	t.Helper()
	origin, _ := url.Parse(wt.Server.URL + "/")
	for _, cookie := range wt.Client.Jar.Cookies(origin) {
		if cookie.Name == "auth" && cookie.Value != "" {
			t.Error("auth cookie should not be set")
		}
	}
}
