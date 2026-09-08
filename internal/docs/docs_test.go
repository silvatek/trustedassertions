package docs

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/PuerkitoBio/goquery"
	"silvatek.uk/trustedassertions/internal/search"
)

func TestDocumentSchemaIsWellFormed(t *testing.T) {
	data, err := os.ReadFile("../../web/static/document.xsd")
	if err != nil {
		t.Fatalf("reading document schema: %v", err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		if _, err := decoder.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("document schema is not well-formed XML: %v", err)
		}
	}
}

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

	if !wordsContains(words, "about", "appear", "begin", "can", "do", "exist", "far", "gl93j73c", "know", "may", "mr", "need", "obvious", "somewhere", "tell", "tester", "truth", "universe", "what") {
		t.Errorf("Unexpected %v", words)
	}

}

func wordsContains(wordList []string, words ...string) bool {
	matches := 0
	for _, w1 := range wordList {
		for _, w2 := range words {
			if w1 == w2 {
				matches++
			}
		}
	}
	return matches == len(words)
}

func TestMetadataVersionCreatedUpdated(t *testing.T) {
	doc, err := MakeDocument(`<document>
	<metadata>
		<title>Dated Doc</title>
		<version>2</version>
		<created>2026-09-07T11:35:23Z</created>
		<updated>2026-09-07T16:17:48Z</updated>
	</metadata>
</document>`)
	if err != nil {
		t.Fatalf("parsing document: %v", err)
	}
	if doc.Metadata.Version != "2" {
		t.Errorf("version = %q, want 2", doc.Metadata.Version)
	}
	if doc.Metadata.Created != "2026-09-07T11:35:23Z" {
		t.Errorf("created = %q, want 2026-09-07T11:35:23Z", doc.Metadata.Created)
	}
	if doc.Metadata.Updated != "2026-09-07T16:17:48Z" {
		t.Errorf("updated = %q, want 2026-09-07T16:17:48Z", doc.Metadata.Updated)
	}
}
