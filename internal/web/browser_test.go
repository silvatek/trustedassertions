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
}
