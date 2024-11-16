package references

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
