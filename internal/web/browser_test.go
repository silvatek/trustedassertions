//go:build browser

package web

import (
	"testing"

	"silvatek.uk/trustedassertions/internal/browsertest"
)

func TestBrowserHome(t *testing.T) {
	b := browsertest.Start(t)
	defer b.Close()

	b.Navigate("/web/home")
	b.WaitVisible("h1")
	b.AssertContains("h1", "Trusted Assertions")

	b.WaitVisible("#searchform")
	b.WaitVisible("#query")
	b.WaitVisible("#submitsearch")

	b.SendKeys("#query", "universe")
	b.Click("#submitsearch")

	b.WaitVisible(".searchresults")
	b.AssertContains(".searchresults", "The universe exists")

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

	b.Click(`#pagemenu a[href^="/web/share"]`)
	b.WaitVisible("#page img")
	b.AssertContains("h2", "Share Item")

	b.Back()
	b.WaitVisible("#category")
	b.AssertContains("h2", "View Assertion")
	b.AssertContains("#subjecttext", "The universe exists")

	b.Click("#issuer")
	b.WaitVisible("#common_name")
	b.AssertContains("h2", "View Entity")
	b.WaitVisible("#references li")
}
