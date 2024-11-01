package assertions

import (
	"crypto/sha256"
	"encoding/xml"
	"os"
	"strings"
)

type Document struct {
	uri      HashUri   `xml:"-"`
	text     string    `xml:"-"`
	XMLName  xml.Name  `xml:"document"`
	Metadata MetaData  `xml:"metadata"`
	Sections []Section `xml:"section"`
}

type MetaData struct {
	XMLName  xml.Name `xml:"metadata"`
	Author   Author   `xml:"author"`
	Title    string   `xml:"title"`
	Keywords string   `xml:"keywords"`
}

type Author struct {
	XMLName xml.Name `xml:"author"`
	Entity  string   `xml:"entity,attr"`
	Name    string   `xml:",chardata"`
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
	Assertion string   `xml:"assertion,attr"`
	Body      string   `xml:",chardata"`
}

func LoadDocument(filename string) (*Document, error) {
	buf, err := os.ReadFile(filename)

	if err == nil {
		return MakeDocument(string(buf))
	} else {
		return nil, err
	}
}

func MakeDocument(content string) (*Document, error) {
	var doc Document

	doc.text = content
	err := xml.Unmarshal([]byte(content), &doc)

	return &doc, err
}

func (d *Document) ParseContent(content string) error {
	d.text = content

	return xml.Unmarshal([]byte(content), d)
}

var DefaultDocumentUri HashUri

func (d *Document) Uri() HashUri {
	if d.uri.IsEmpty() {
		hash := sha256.New()
		hash.Write([]byte(d.text))
		d.uri = MakeUriB(hash.Sum(nil), "document")
	}
	return d.uri
}

func (d Document) Type() string {
	return "Document"
}

func (d Document) Content() string {
	return d.text
}

func (d Document) Summary() string {
	return d.Metadata.Title
}

func (d Document) References() []HashUri {
	refs := make([]HashUri, 0)
	if d.Metadata.Author.Entity != "" {
		refs = append(refs, UriFromString(d.Metadata.Author.Entity))
	}
	for _, span := range d.allAssertions() {
		refs = append(refs, UriFromString(span.Assertion))
	}
	return refs
}

func (d *Document) allAssertions() []Span {
	assertions := make([]Span, 0)

	for _, sect := range d.Sections {
		for _, para := range sect.Paragraphs {
			for _, span := range para.Spans {
				if span.Assertion != "" {
					assertions = append(assertions, span)
				}
			}
		}
	}

	return assertions
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
				if span.Assertion != "" {
					uri := UriFromString(span.Assertion)
					html = html + "<a href='" + uri.WebPath() + "'>" + span.Body + "</a>"
				} else if span.Statement != "" {
					uri := UriFromString(span.Statement)
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

func (doc *Document) TextContent() string {
	var sb strings.Builder

	sb.WriteString(doc.Metadata.Title)
	sb.WriteString(" ")
	sb.WriteString(doc.Metadata.Author.Name)
	sb.WriteString(" ")
	sb.WriteString(doc.Metadata.Keywords)
	sb.WriteString(" ")

	for _, sect := range doc.Sections {
		if sect.Title != nil {
			sb.WriteString(sect.Title.Text)
			sb.WriteString(" ")
		}
		for _, para := range sect.Paragraphs {
			for _, span := range para.Spans {
				sb.WriteString(span.Body)
				sb.WriteString(" ")
			}
		}
	}
	return sb.String()
}
