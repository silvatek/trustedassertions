package web

import (
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

func TestStaticFilesSurviveWriteTimeout(t *testing.T) {
	TemplateDir = "../../web"

	r := mux.NewRouter()
	r.PathPrefix("/web/static").Handler(StaticHandler())
	protect := csrf.Protect(
		[]byte("default_csrf_key"),
		csrf.SameSite(csrf.SameSiteStrictMode),
		csrf.FieldName("authenticity_token"),
		csrf.Path("/"),
		csrf.CookieName("authenticity_token"),
	)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{
		Handler:      protect(r),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	base := "http://" + ln.Addr().String()
	files := []string{"default.css", "TickSpeech.svg", "document.xsd"}
	for _, name := range files {
		want, err := os.ReadFile(filepath.Join(TemplateDir, "static", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		resp, err := http.Get(base + "/web/static/" + name)
		if err != nil {
			t.Fatalf("GET %s: %v", name, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatalf("read body %s: %v", name, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s status %d, want 200", name, resp.StatusCode)
		}
		if len(body) != len(want) {
			t.Errorf("%s got %d bytes, want %d", name, len(body), len(want))
		}
	}
}
