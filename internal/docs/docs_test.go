package docs

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/PuerkitoBio/goquery"
	"silvatek.uk/trustedassertions/internal/search"
)

func TestTestPoc1(t *testing.T) {
	doc, err := LoadDocument("../../testdata/documents/testdoc1.xml")

	if err != nil {
		t.Errorf("Error loading/parsing document: %v", err)
		return
	}

	docHtml := doc.ToHtml()

	tmpl, _ := template.ParseFiles("./test.html")

	var buf bytes.Buffer

	tmpl.ExecuteTemplate(&buf, "test.html", docHtml)

	html, err := goquery.NewDocumentFromReader(&buf)
	if err != nil {
		t.Errorf("Error parsing html: %v", err)
	}

	if !strings.Contains(html.Find("h1").Text(), "About the Universe") {
		t.Error("Did not find expected title")
	}

	if html.Find("a").Text() != "The universe exists" {
		t.Error("Did not find expected hyperlink")
	}

	words := search.SearchWords(doc.TextContent())

	if !search.WordsContains(words, "about", "appear", "begin", "can", "do", "exist", "far", "gl93j73c", "know", "may", "mr", "need", "obvious", "somewhere", "tell", "tester", "truth", "universe", "what") {
		t.Errorf("Unexpected %v", words)
	}

}
