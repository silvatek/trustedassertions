package docs

import (
	"encoding/xml"
	"os"

	"silvatek.uk/trustedassertions/internal/assertions"
)

type Document struct {
	XMLName  xml.Name  `xml:"document"`
	Metadata MetaData  `xml:"metadata"`
	Sections []Section `xml:"section"`
}

type MetaData struct {
	XMLName xml.Name `xml:"metadata"`
}

type Section struct {
	XMLName    xml.Name    `xml:"section"`
	Attrs      []xml.Attr  `xml:",any,attr"`
	Title      *Title      `xml:"title"`
	Paragraphs []Paragraph `xml:"paragraph"`
}

type Title struct {
	XMLName xml.Name `xml:"title"`
	Text    string   `xml:",chardata"`
}

type Paragraph struct {
	XMLName xml.Name   `xml:"paragraph"`
	Attrs   []xml.Attr `xml:",any,attr"`
	Spans   []Span     `xml:"span"`
}

type Span struct {
	XMLName   xml.Name `xml:"span"`
	Statement string   `xml:"statement,attr"`
	Body      string   `xml:",chardata"`
}

func LoadDocument(filename string) (*Document, error) {
	var doc Document

	text, err := os.ReadFile(filename)

	if err == nil {
		err = xml.Unmarshal([]byte(text), &doc)
	}

	return &doc, err
}

func (doc *Document) ToHtml() string {
	var html string

	for _, sect := range doc.Sections {
		html = html + "\n<div class='docsection'>"
		if sect.Title != nil {
			html = html + "\n   <h1>" + sect.Title.Text + "</h1>"
		}
		for _, para := range sect.Paragraphs {
			html = html + "\n   <div class='docpara'>\n      "
			for _, span := range para.Spans {
				if span.Statement != "" {
					uri := assertions.UriFromString(span.Statement)
					html = html + "<a href='" + uri.WebPath() + "'>" + span.Body + "</a>"
				} else {
					html = html + "<span>" + span.Body + "</span>"
				}
			}
			html = html + "\n   </div>"
		}
		html = html + "\n</div>"
	}
	return html
}
