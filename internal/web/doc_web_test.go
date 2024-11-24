package web

import (
	"context"
	"testing"

	"silvatek.uk/trustedassertions/internal/datastore"
)

func TestViewDoc(t *testing.T) {
	wt := NewWebTest(t)
	defer wt.Close()

	docs, _ := datastore.ActiveDataStore.Search(context.Background(), "GL93J73C")
	docHash := docs[0].Uri.Hash()

	page := wt.GetPage("/web/documents/" + docHash)
	page.AssertHtmlQuery("h2", "View Document")
}
