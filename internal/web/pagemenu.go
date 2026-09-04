package web

import (
	"strings"

	"silvatek.uk/trustedassertions/internal/auth"
)

type PageMenu struct {
	Items []PageMenuItem
}

type PageMenuItem struct {
	Menu         *PageMenu
	Text         string
	Target       string
	Separator    string
	Style        string
	RequiresRole string
}

func (i PageMenuItem) IsLink() bool {
	return i.Target != ""
}

// VisibleTo is true when RequiresRole is empty, or when the user has that role.
func (i PageMenuItem) VisibleTo(user *auth.User) bool {
	if i.RequiresRole == "" {
		return true
	}
	if user == nil {
		return false
	}
	return user.HasRole(i.RequiresRole)
}

// UseHtmx is false for API/raw links so the browser loads them as full documents.
func (i PageMenuItem) UseHtmx() bool {
	return i.IsLink() && !strings.HasPrefix(i.Target, "/api")
}

func (m *PageMenu) AddItem(item *PageMenuItem) {
	if m.Items == nil {
		m.Items = make([]PageMenuItem, 0)
	}

	n := len(m.Items)
	if n > 0 {
		item.Separator = "|"
	}

	m.Items = append(m.Items, *item)
}

func (m *PageMenu) AddLink(text string, target string) {
	item := PageMenuItem{
		Menu:   m,
		Text:   text,
		Target: target,
		Style:  "leftlink",
	}
	m.AddItem(&item)
}

func (m *PageMenu) AddRightLink(text string, target string) {
	item := PageMenuItem{
		Menu:   m,
		Text:   text,
		Target: target,
		Style:  "rightlink",
	}
	m.AddItem(&item)
}

func (m *PageMenu) AddRightText(text string) {
	item := PageMenuItem{
		Menu:  m,
		Text:  text,
		Style: "rightlink",
	}
	m.AddItem(&item)
}
