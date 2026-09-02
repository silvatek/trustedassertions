package web

import "testing"

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
