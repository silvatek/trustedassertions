package web

import (
	"net/http"

	"github.com/gorilla/mux"
	"silvatek.uk/trustedassertions/internal/appcontext"
)

// Add handlers for URL paths only used by attackers
func AddAttackHandlers(r *mux.Router) {
	// Specific filenames
	for _, path := range []string{
		"/config.json", "/web.config", "/wp-config.php", "/api/.env", "/cloud-config.yml",
		"/docker-compose.yml", "/user_secrets.yml", "/secrets.json",
		"/database.sql", "/backup.sql", "/backup.zip", "/backup.tar.gz",
		"/config.php", "/config/database.php", "/server-status", "/phpinfo.php", "/wp-config.php", "/config/database.php", "/config/production.json",
	} {
		r.HandleFunc(path, AttackHandler)
	}

	// Entire directories
	for _, prefix := range []string{
		"/etc/", "/.ssh/", "/.git/", "/.svn", "/_vti_pvt/", "/.vscode/", "/.kube", "/.aws",
	} {
		r.PathPrefix(prefix).HandlerFunc(AttackHandler)
	}

	r.HandleFunc("/robots.txt", RobotsTxtHandler)
}

func AttackHandler(w http.ResponseWriter, r *http.Request) {
	log.DebugfX(appcontext.NewWebContext(r), "Dropping suspect request: %v", r.URL)
	w.WriteHeader(404)
}

func RobotsTxtHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("User-agent: *\n"))
	w.Write([]byte("Disallow: /\n"))
	w.Write([]byte("Allow: /robots.txt\n"))
}
