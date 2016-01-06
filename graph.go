package gta

// Graph is an adjacency list representation of a graph using maps.
type Graph struct {
	graph map[string]map[string]bool
}

// Traverse is a a simple recurisve depth first traversal of a directed cyclic graph.
func (g *Graph) Traverse(node string, ref *map[string]bool) {
	mark := *ref
	// we've already visited this node
	if visited, ok := mark[node]; visited && ok {
		return
	}
	// we mark the node as visited
	mark[node] = true

	if edges, ok := g.graph[node]; ok {
		for edge := range edges {
			g.Traverse(edge, ref)
		}
	}

	return
}
