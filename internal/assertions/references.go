package assertions

type Referenceable interface {
	Uri() HashUri
	Type() string
	Content() string
}

type Reference struct {
	target  HashUri
	refType string
	source  HashUri
}
