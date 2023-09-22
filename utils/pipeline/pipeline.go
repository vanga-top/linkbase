package pipeline

type Pipeline interface {
	Add(node ...Node)
}
