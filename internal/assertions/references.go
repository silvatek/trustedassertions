package assertions

type Referenceable interface {
	Uri() string
	Type() string
	Content() string
}

type Reference struct {
	target  string
	refType string
	source  string
}
