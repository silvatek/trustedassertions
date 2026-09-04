package web

import (
	"net/http"
	"net/url"

	"silvatek.uk/trustedassertions/internal/appcontext"
	"silvatek.uk/trustedassertions/internal/auth"
	"silvatek.uk/trustedassertions/internal/datastore"
)

type adminPageData struct {
	CreatedCode   string
	Registrations []auth.Registration
}

func requireAdministrator(w http.ResponseWriter, r *http.Request) bool {
	ctx := appcontext.NewWebContext(r)
	username := authUsername(r)
	if username == "" {
		HandleError(ctx, ErrorNoAuth, w, r)
		return false
	}

	user, err := datastore.ActiveDataStore.FetchUser(ctx, username)
	if err != nil {
		HandleError(ctx, ErrorUserNotFound.instance("User not found: "+username), w, r)
		return false
	}

	if !user.HasRole(auth.RoleAdministrator) {
		HandleError(ctx, ErrorForbidden.instance("User is not an administrator: "+username), w, r)
		return false
	}

	return true
}

func AdminWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	if !requireAdministrator(w, r) {
		return
	}

	if r.Method != http.MethodGet {
		http.Redirect(w, r, "/web/admin", http.StatusSeeOther)
		return
	}

	regs, err := datastore.ActiveDataStore.ListRegistrations(ctx)
	if err != nil {
		HandleError(ctx, ErrorCreateInvite.instance("Error listing registrations"), w, r)
		return
	}

	RenderWebPage(ctx, "admin", adminPageData{
		CreatedCode:   r.URL.Query().Get("created"),
		Registrations: regs,
	}, nil, w, r)
}

func AdminInvitesWebHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appcontext.NewWebContext(r)
	if !requireAdministrator(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/web/admin", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	code := auth.GenerateInviteCode()

	reg := auth.Registration{
		Code:   code,
		Status: "Pending",
		Roles:  rolesFromInviteForm(r),
	}
	if err := datastore.ActiveDataStore.StoreRegistration(ctx, reg); err != nil {
		HandleError(ctx, ErrorCreateInvite.instance("Error storing invitation"), w, r)
		return
	}

	http.Redirect(w, r, "/web/admin?created="+url.QueryEscape(code), http.StatusSeeOther)
}

func rolesFromInviteForm(r *http.Request) []string {
	roles := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, value := range r.Form["role"] {
		if value != auth.RoleAuthor && value != auth.RoleAdministrator {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		roles = append(roles, value)
	}
	return roles
}
