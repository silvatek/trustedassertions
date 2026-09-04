package web

import (
	"testing"

	"silvatek.uk/trustedassertions/internal/auth"
)

func TestBasicPageMenu(t *testing.T) {
	menu := PageMenu{}
	menu.AddLink("Link 1", "/link1")
	menu.AddRightLink("Link 2", "/link2")
	menu.AddRightText("Text 1")

	if len(menu.Items) != 3 {
		t.Errorf("Unexpected menu item count: %d", len(menu.Items))
	}

	if !menu.Items[0].UseHtmx() {
		t.Errorf("Web menu link should use htmx")
	}

	raw := PageMenuItem{Text: "Raw", Target: "/api/v1/statements/abc"}
	if raw.UseHtmx() {
		t.Errorf("API menu link should not use htmx")
	}
}

func TestPageMenuItemVisibleTo(t *testing.T) {
	open := PageMenuItem{Text: "Home", Target: "/"}
	gated := PageMenuItem{Text: "Admin", Target: "/web/admin", RequiresRole: auth.RoleAdministrator}

	if !open.VisibleTo(nil) {
		t.Errorf("item with empty RequiresRole should be visible with no user")
	}

	author := &auth.User{Id: "author"}
	author.AddRole(auth.RoleAuthor)
	if !open.VisibleTo(author) {
		t.Errorf("item with empty RequiresRole should be visible to any user")
	}

	if gated.VisibleTo(nil) {
		t.Errorf("role-gated item should not be visible with no user")
	}
	if gated.VisibleTo(author) {
		t.Errorf("role-gated item should not be visible without the required role")
	}

	admin := &auth.User{Id: "admin"}
	admin.AddRole(auth.RoleAdministrator)
	if !gated.VisibleTo(admin) {
		t.Errorf("role-gated item should be visible when the user has the required role")
	}

	menu := PageMenu{}
	menu.AddItem(&open)
	menu.AddItem(&gated)
	visible := menu.VisibleItems(author)
	if len(visible.Items) != 1 || visible.Items[0].Text != "Home" {
		t.Errorf("VisibleItems should omit gated items the user cannot see, got %+v", visible.Items)
	}
	if visible.Items[0].Separator != "" {
		t.Errorf("first visible item should not keep a leftover separator")
	}
}
