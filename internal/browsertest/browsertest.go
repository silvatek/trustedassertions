//go:build browser

package browsertest

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const defaultBaseURL = "http://127.0.0.1:8080"
const testTimeout = 45 * time.Second
const preflightTimeout = 3 * time.Second

// Target is the server the browser tests drive. BROWSER_BASE_URL overrides
// the local default so the same tests can run against a live instance.
func Target() string {
	base := strings.TrimSpace(os.Getenv("BROWSER_BASE_URL"))
	if base == "" {
		base = defaultBaseURL
	}
	return strings.TrimRight(base, "/")
}

// Browser is a headless Chrome session pointed at Target.
type Browser struct {
	t       *testing.T
	ctx     context.Context
	cancels []context.CancelFunc
	base    string
	closed  bool
}

// Start preflights Target, launches Chrome, and applies a per-test timeout.
func Start(t *testing.T) *Browser {
	t.Helper()
	base := Target()
	preflight(t, base)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1280, 800),
	)
	if os.Getenv("BROWSER_HEADFUL") == "1" {
		opts = append(opts, chromedp.Flag("headless", false))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)
	ctx, timeoutCancel := context.WithTimeout(ctx, testTimeout)

	b := &Browser{
		t:       t,
		ctx:     ctx,
		cancels: []context.CancelFunc{timeoutCancel, ctxCancel, allocCancel},
		base:    base,
	}

	if err := chromedp.Run(ctx, chromedp.Navigate("about:blank")); err != nil {
		b.Close()
		if chromeMissing(err) {
			t.Fatalf("Chrome/Chromium not found; install Chrome or Chromium and ensure it is on PATH")
		}
		t.Fatalf("start Chrome: %v", err)
	}

	t.Cleanup(b.Close)
	return b
}

func preflight(t *testing.T, base string) {
	t.Helper()
	client := &http.Client{Timeout: preflightTimeout}
	resp, err := client.Get(base)
	if err != nil {
		t.Fatalf("start the server or set BROWSER_BASE_URL: nothing listening at %s (%v)", base, err)
	}
	resp.Body.Close()
}

func chromeMissing(err error) bool {
	s := err.Error()
	return strings.Contains(s, "executable file not found") ||
		strings.Contains(s, "chrome not found") ||
		strings.Contains(s, "chromium not found")
}

func (b *Browser) resolve(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	u, err := url.Parse(b.base)
	if err != nil {
		return b.base + "/" + strings.TrimLeft(path, "/")
	}
	return u.JoinPath(path).String()
}

func (b *Browser) run(actions ...chromedp.Action) {
	b.t.Helper()
	if err := chromedp.Run(b.ctx, actions...); err != nil {
		if chromeMissing(err) {
			b.t.Fatalf("Chrome/Chromium not found; install Chrome or Chromium and ensure it is on PATH")
		}
		b.t.Fatalf("%v", err)
	}
}

func (b *Browser) Navigate(path string) {
	b.t.Helper()
	b.run(chromedp.Navigate(b.resolve(path)))
}

func (b *Browser) Click(sel string) {
	b.t.Helper()
	b.run(chromedp.Click(sel, chromedp.ByQuery))
}

// ClickLinkNextTo clicks the first link in the table cell after a cell whose
// text is exactly text (the URI link beside a search-result summary).
func (b *Browser) ClickLinkNextTo(text string) {
	b.t.Helper()
	xpath := `//td[normalize-space()=` + xpathLiteral(text) + `]/following-sibling::td//a`
	b.run(chromedp.Click(xpath, chromedp.BySearch))
}

func xpathLiteral(s string) string {
	if !strings.Contains(s, "'") {
		return "'" + s + "'"
	}
	return `"` + s + `"`
}

func (b *Browser) SendKeys(sel, keys string) {
	b.t.Helper()
	b.run(chromedp.SendKeys(sel, keys, chromedp.ByQuery))
}

func queryBy(sel string) chromedp.QueryOption {
	if strings.HasPrefix(sel, "//") {
		return chromedp.BySearch
	}
	return chromedp.ByQuery
}

func (b *Browser) WaitVisible(sel string) {
	b.t.Helper()
	b.run(chromedp.WaitVisible(sel, queryBy(sel)))
}

func (b *Browser) Text(sel string) string {
	b.t.Helper()
	var text string
	b.run(chromedp.Text(sel, &text, queryBy(sel)))
	return text
}

func (b *Browser) AssertContains(sel, want string) {
	b.t.Helper()
	got := b.Text(sel)
	if !strings.Contains(got, want) {
		b.t.Errorf("selector %s: want substring %q, got %q", sel, want, got)
	}
}

func (b *Browser) Close() {
	if b.closed {
		return
	}
	b.closed = true
	for _, cancel := range b.cancels {
		cancel()
	}
}
