package web

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
)

// Add handlers for URL paths only used by attackers
func AddAttackHandlers(r *mux.Router) {
	// Specific filenames
	for _, path := range []string{
		"/config.json", "/web.config", "/wp-config.php", "/api/.env", "/cloud-config.yml", "/config/production.json", "/feed/", "/.DS_Store", "/about", "/server",
		"/.env", "/.env~", "/.env.dev", "/.env.local", "/.env.example", "/admin/.env", "/_all_dbs", "/login.action",
		"/docker-compose.yml", "/user_secrets.yml", "/secrets.json",
		"/database.sql", "/backup.sql", "/backup.zip", "/backup.tar.gz",
		"/index.php", "/config.php", "/config/database.php", "/server-status", "/phpinfo.php", "/wp-config.php", "/config/database.php", "/xmlrpc.php?rsd",
	} {
		r.HandleFunc(path, AttackHandler)
	}

	// Entire directories
	for _, prefix := range []string{
		"/etc/", "/.ssh/", "/.git/", "/.svn", "/_vti_pvt/", "/.vscode/", "/.kube", "/.aws", "/.docker/", "/ecp/", "/admin/", "/debug/", "/_/", "/cgi-bin/",
	} {
		r.PathPrefix(prefix).HandlerFunc(AttackHandler)
	}

	r.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return strings.Contains(r.URL.Path, "/wp-includes/")
	}).HandlerFunc(AttackHandler)

	r.MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		return strings.Contains(r.Host, "34.160.")
	}).HandlerFunc(AttackHandler)

	r.HandleFunc("/robots.txt", RobotsTxtHandler)
}

// Respond with a 404 status code and no body
func AttackHandler(w http.ResponseWriter, r *http.Request) {
	log.DebugfX(appcontext.NewWebContext(r), "Dropping suspect request: %v", r.URL)
	SetCacheControl(w, 7*24*60*60)
	//w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(410)
}

func RobotsTxtHandler(w http.ResponseWriter, r *http.Request) {
	SetCacheControl(w, 4*60*60)
	w.WriteHeader(200)
	w.Write([]byte("User-agent: *\n"))
	w.Write([]byte("Disallow: /\n"))
	w.Write([]byte("Allow: /robots.txt\n"))
}
