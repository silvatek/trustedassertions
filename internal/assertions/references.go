package assertions

type Referenceable interface {
	Uri() HashUri
	Type() string
	Content() string
}
