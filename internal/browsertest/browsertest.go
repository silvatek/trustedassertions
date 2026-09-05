//go:build browser

package browsertest

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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
const waitTimeout = 10 * time.Second
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
	docMark string
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

func (b *Browser) waitCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(b.ctx, waitTimeout)
}

func (b *Browser) run(actions ...chromedp.Action) {
	b.t.Helper()
	ctx, cancel := b.waitCtx()
	defer cancel()
	if err := chromedp.Run(ctx, actions...); err != nil {
		if chromeMissing(err) {
			b.t.Fatalf("Chrome/Chromium not found; install Chrome or Chromium and ensure it is on PATH")
		}
		b.t.Fatal(b.waitErr(err))
	}
}

func (b *Browser) waitErr(err error) string {
	if b.ctx.Err() != nil {
		return fmt.Sprintf("browser session timed out after %s: %v", testTimeout, b.ctx.Err())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Sprintf("timed out after %s: %v", waitTimeout, err)
	}
	return err.Error()
}

func (b *Browser) Navigate(path string) {
	b.t.Helper()
	b.run(chromedp.Navigate(b.resolve(path)))
}

func (b *Browser) Click(sel string) {
	b.t.Helper()
	b.run(chromedp.Click(sel, chromedp.ByQuery))
}

func (b *Browser) ClickMenu(text string) {
	b.t.Helper()
	xpath := `//div[@id='pagemenu']//a[normalize-space()=` + xpathLiteral(text) + `]`
	b.run(chromedp.Click(xpath, chromedp.BySearch))
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

func (b *Browser) WaitVisible(sels ...string) {
	b.t.Helper()
	for _, sel := range sels {
		b.run(chromedp.WaitVisible(sel, queryBy(sel)))
	}
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
	if want == "" {
		if got != "" {
			b.t.Errorf("selector %s: want empty, got %q", sel, got)
		}
		return
	}
	if !strings.Contains(got, want) {
		b.t.Errorf("selector %s: want substring %q, got %q", sel, want, got)
	}
}

func (b *Browser) WaitContains(sel, want string) {
	b.t.Helper()
	ctx, cancel := b.waitCtx()
	defer cancel()
	var got string
	for {
		err := chromedp.Run(ctx, chromedp.Text(sel, &got, queryBy(sel)))
		if err == nil && strings.Contains(got, want) {
			return
		}
		if ctx.Err() != nil {
			b.t.Fatalf("selector %s: want substring %q, got %q: %s", sel, want, got, b.waitErr(ctx.Err()))
		}
		select {
		case <-ctx.Done():
			b.t.Fatalf("selector %s: want substring %q, got %q: %s", sel, want, got, b.waitErr(ctx.Err()))
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (b *Browser) Back() {
	b.t.Helper()
	var unused any
	b.run(chromedp.Evaluate(`window.history.back()`, &unused))
}

func (b *Browser) MarkDocument() {
	b.t.Helper()
	b.docMark = fmt.Sprintf("%d", rand.Int())
	var unused any
	b.run(chromedp.Evaluate(
		fmt.Sprintf(`document.documentElement.setAttribute('data-ta-doc', %q)`, b.docMark),
		&unused,
	))
}

func (b *Browser) AssertSameMarkedDocument() {
	b.t.Helper()
	if b.docMark == "" {
		b.t.Fatal("AssertSameMarkedDocument called before MarkDocument")
	}
	var got string
	b.run(chromedp.Evaluate(`document.documentElement.getAttribute('data-ta-doc')`, &got))
	if got != b.docMark {
		b.t.Errorf("expected an HTMX fragment swap on the existing page, but the document was replaced")
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
