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
	b.WaitVisible("#searchform", "#query", "#submitsearch")

	b.SendKeys("#query", "universe")
	b.ClickHtmx("#submitsearch", ".searchresults")

	b.ClickLinkNextTo("The universe exists")
	b.WaitVisible("#content")
	b.Click("#references a")
	b.WaitVisible("#category")

	b.ClickMenu("Share")
	b.WaitVisible("#page img")
	b.Back()
	b.WaitVisible("#category")

	b.ClickMenu("Home")
	b.WaitVisible("#searchform")
}

func TestBrowserRegister(t *testing.T) {
	code := browsertest.RequireRegCode(t)
	userID, password := browsertest.NewTestUser()
	entityName := "Browser test entity " + userID
	statementText := "Browser test statement " + userID

	b := browsertest.StartRegister(t)
	defer b.Close()

	b.NavigateHome()
	b.WaitVisible("#searchform")
	b.ClickMenuHtmx("Register", "#reg_code", "#user_id", "#password1", "#password2", "#register")
	b.Fill("#reg_code", code, "#user_id", userID, "#password1", password, "#password2", password)
	b.Click("#register")

	b.WaitVisible("#user_id", "#password", "#login")
	b.Fill("#user_id", userID, "#password", password)
	b.Click("#login")
	b.WaitVisible("#searchform")

	b.ClickLinkHtmx("Entity with signing key", "#commonname", "#submit")
	b.SendKeys("#commonname", entityName)
	b.ClickHtmx("#submit", "#common_name")
	b.AssertContains("#common_name", entityName)

	b.ClickMenu("Home")
	b.WaitVisible("#searchform")
	b.ClickLinkHtmx("Statement and Assertion", "#statement", "#sign_as", "#submit")
	b.RequireCount("#sign_as option")
	b.SendKeys("#statement", statementText)
	b.ClickHtmx("#submit", "#subjecttext")
	b.AssertContains("#subjecttext", statementText)

	if !b.HasMenuLink("Admin") {
		return
	}
	b.ClickMenuHtmx("Admin", "#create-invite")
	b.Click("#create-invite")
	b.WaitVisible("#created-invite", ".invite-code")
	b.AssertNonEmpty(".invite-code")
}
