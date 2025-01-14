package references

import "errors"

type Reference struct {
	Source  HashUri // The source has a reference to the target
	Target  HashUri
	Summary string
}

// Referenceable is a core data type that can be referenced by an assertion.
type Referenceable interface {
	Uri() HashUri
	Type() string
	Content() string
	Summary() string
	TextContent() string
	References() []HashUri
	ParseContent(content string) error
}

// A map of URIs to the Referenceables they refer to
type ReferenceMap map[HashUri]Referenceable

var REF_ERROR ReferenceError

type ReferenceError struct {
}

func (e ReferenceError) Uri() HashUri {
	return ERROR_URI
}

func (e ReferenceError) Type() string {
	return "ERROR"
}

func (e ReferenceError) Content() string {
	return "ERROR"
}

func (e ReferenceError) Summary() string {
	return "ERROR"
}

func (e ReferenceError) TextContent() string {
	return "ERROR"
}

func (e ReferenceError) References() []HashUri {
	return make([]HashUri, 0)
}

func (e ReferenceError) ParseContent(content string) error {
	return errors.New("Error URI cannot parse content")
}

func (e ReferenceError) Error() string {
	return "Reference error"
}
