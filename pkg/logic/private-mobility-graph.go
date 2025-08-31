/*

implements an undirected graph that represents the private
mobility graph.

the graph is represented as an edge set

*/

package logic

import (
	"math/rand"
)
	

type Edge struct {
	District1 int
	District2 int
}

type PrivateMobilityGraph struct {
	Map map[Edge]struct{}

	// k top frequently visited districts
	// k int

	// number of districts 
	DistrictsAmount int
}

// initializes an empty graph
func NewPrivateMobilityGraph(DistrictsAmount int) *PrivateMobilityGraph {
	return &PrivateMobilityGraph{
		Map: make(map[Edge]struct{}),
		// k:   k,
		DistrictsAmount: DistrictsAmount,
	}
}

func (g* PrivateMobilityGraph) AddKDistricts(districts []int) {
	// for all combinations of districts, add an edge
	for i := 0; i < len(districts); i++ {
		for j := i + 1; j < len(districts); j++ {
			g.AddEdge(districts[i], districts[j])
		}
	}
}

// adds one edge to the graph
func (g *PrivateMobilityGraph) AddEdge(district1, district2 int) {
	edge := Edge{District1: district1, District2: district2}
	// check if the edge already exists, don't add again
	if g.HasEdge(district1, district2) {
		return
	}
	// add the edge to the graph
	g.Map[edge] = struct{}{}
}

// returns all edges in the graph as a slice
func (g *PrivateMobilityGraph) GetEdges() []Edge {
	edges := make([]Edge, 0, len(g.Map))
	for edge := range g.Map {
		edges = append(edges, edge)
	}
	return edges
}

// checks if an edge exists in the graph 
func (g *PrivateMobilityGraph) HasEdge(district1, district2 int) bool {
    edge1 := Edge{District1: district1, District2: district2}
    edge2 := Edge{District1: district2, District2: district1}

	if g.Map == nil {
		return false
	}
    _, exists1 := g.Map[edge1]
    _, exists2 := g.Map[edge2]

    return exists1 || exists2
}

// removes an edge from the graph
func (g *PrivateMobilityGraph) RemoveEdge(district1, district2 int) {
	edge := Edge{District1: district1, District2: district2}
	delete(g.Map, edge)
}

// privatizes the graph 
// for each possible combination of districts, if an edge 
// already exists we remove it with probability 1-p, if an 
// edge does not exist we add it with probability 1-p  
func (g *PrivateMobilityGraph) Privatize(p float32) {
	for i := 0; i < g.DistrictsAmount; i++ {
		for j := i + 1; j < g.DistrictsAmount; j++ {
			// for each possible combination of districts 
			if rand.Float32() < 1-p {
				if g.HasEdge(i, j) {
					g.RemoveEdge(i, j)
				} else {
					g.AddEdge(i, j)
				}
			}
		}
	}

}

/*

// ToBitVector converts the private mobility graph into a bit vector.
func (g *PrivateMobilityGraph) ToBitVector() []byte {
    // Calculate the total number of possible edges
    totalEdges := g.DistrictsAmount * (g.DistrictsAmount - 1) / 2

    // Create a bit vector with enough bytes to hold all bits
    bitVector := make([]byte, (totalEdges+7)/8) // +7 ensures rounding up to the nearest byte

    // Map each edge to a unique index in the bit vector
    index := 0
    for i := 0; i < g.DistrictsAmount; i++ {
        for j := i + 1; j < g.DistrictsAmount; j++ {
            if g.HasEdge(i, j) {
                // Set the corresponding bit in the bit vector
                byteIndex := index / 8
                bitIndex := index % 8
                bitVector[byteIndex] |= (1 << bitIndex)
            }
            index++
        }
    }

    return bitVector
}

// FromBitVector converts a bit vector back into a private mobility graph.
func FromBitVector(bitVector []byte, districtsAmount int) *PrivateMobilityGraph {
    // Initialize a new private mobility graph
    graph := NewPrivateMobilityGraph()
    graph.DistrictsAmount = districtsAmount

    // Map each bit in the bit vector back to an edge
    index := 0
    for i := 0; i < districtsAmount; i++ {
        for j := i + 1; j < districtsAmount; j++ {
            byteIndex := index / 8
            bitIndex := index % 8
            if bitVector[byteIndex]&(1<<bitIndex) != 0 {
                graph.AddEdge(i, j)
            }
            index++
        }
    }

    return graph
}

*/