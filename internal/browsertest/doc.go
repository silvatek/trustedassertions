// Package browsertest drives headless Chrome against a running server.
package browsertest

// Keep chromedp required when go mod tidy runs without the browser build tag.
import _ "github.com/chromedp/chromedp"
