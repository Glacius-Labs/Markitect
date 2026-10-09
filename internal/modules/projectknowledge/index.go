package projectknowledge

import "encoding/json"

func (i *Index) Binding() Binding { return i.binding }
func (i *Index) Nodes() []Node {
	out := make([]Node, len(i.nodes))
	for n, v := range i.nodes {
		out[n] = cloneNode(v)
	}
	return out
}
func (i *Index) Edges() []Edge { return append([]Edge(nil), i.edges...) }
func (i *Index) Node(id string) (Node, error) {
	n, ok := i.byID[id]
	if !ok {
		return Node{}, ErrNotFound
	}
	return cloneNode(i.nodes[n]), nil
}
func (i *Index) Outgoing(id string) ([]Edge, error) {
	if _, ok := i.byID[id]; !ok {
		return nil, ErrNotFound
	}
	return i.edgesAt(i.out[id]), nil
}
func (i *Index) Incoming(id string) ([]Edge, error) {
	if _, ok := i.byID[id]; !ok {
		return nil, ErrNotFound
	}
	return i.edgesAt(i.in[id]), nil
}
func (i *Index) edgesAt(ids []int) []Edge {
	out := make([]Edge, 0, len(ids))
	for _, n := range ids {
		out = append(out, i.edges[n])
	}
	return out
}
func cloneNode(n Node) Node {
	c := n
	c.Properties = make(map[string]json.RawMessage, len(n.Properties))
	for k, v := range n.Properties {
		c.Properties[k] = append(json.RawMessage(nil), v...)
	}
	if n.Source != nil {
		s := *n.Source
		c.Source = &s
	}
	return c
}
