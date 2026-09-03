package web

import (
	"context"
	"encoding/json"
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
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
