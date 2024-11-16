package references

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
