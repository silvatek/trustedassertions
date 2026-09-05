//go:build browser

package web

import (
	"testing"

	"silvatek.uk/trustedassertions/internal/browsertest"
)

func TestBrowserHome(t *testing.T) {
	b := browsertest.Start(t)
	defer b.Close()

	b.NavigateHome()
	b.WaitVisible("h1", "#searchform", "#query", "#submitsearch")
	b.AssertContains("h1", "Trusted Assertions")
	b.MarkDocument()

	b.SendKeys("#query", "universe")
	b.Click("#submitsearch")

	b.WaitVisible(".searchresults")
	b.AssertContains(".searchresults", "The universe exists")
	b.AssertSameMarkedDocument()

	b.ClickLinkNextTo("The universe exists")
	b.WaitVisible("#content")
	b.AssertContains("h2", "View Statement")
	b.AssertContains("#content", "The universe exists")

	b.WaitVisible("#references li")
	b.AssertContains("h3", "References")
	b.Click("#references a")
	b.WaitVisible("#category")
	b.AssertContains("h2", "View Assertion")
	b.AssertContains("#subjecttext", "The universe exists")

	b.ClickMenu("Share")
	b.WaitVisible("#page img")
	b.AssertContains("h2", "Share Item")

	b.Back()
	b.WaitVisible("#category")
	b.AssertContains("h2", "View Assertion")
	b.AssertContains("#subjecttext", "The universe exists")

	b.Click("#issuer")
	b.WaitVisible("#common_name", "#references li")
	b.AssertContains("h2", "View Entity")

	b.ClickMenu("Home")
	b.WaitVisible("#searchform")

	b.ClickMenu("Register")
	b.WaitVisible("#reg_code", "#user_id", "#password1", "#password2", "#register")
	b.AssertContains(".error", "")

	b.SendKeys("#reg_code", "not-a-valid-code")
	b.SendKeys("#user_id", "browser-test-user")
	b.SendKeys("#password1", "dummy-password-123")
	b.SendKeys("#password2", "dummy-password-123")
	b.Click("#register")

	b.WaitContains(".error", "Registration code not valid")
	b.WaitVisible("#reg_code")

	b.ClickMenu("Login")
	b.WaitVisible("#user_id", "#password", "#login")
	b.AssertContains("h2", "Login")
	b.AssertContains(".error", "")

	b.SendKeys("#user_id", "browser-test-user")
	b.SendKeys("#password", "dummy-password-123")
	b.Click("#login")

	b.WaitContains(".error", "Unable to verify identity")
	b.WaitVisible("#login")

	b.Click("#pagelogo")
	b.WaitVisible("#searchform")
	b.AssertContains("h1", "Trusted Assertions")
}
