/*

implements an undirected graph that represents the global
mobility graph.

the graph is represented as an edge set

*/

package logic

import (
	model "marathon-sim/datamodel"
	"math/rand"
)

type GlobalMobilityGraph struct {
	Map map[Edge]float32

	// k top frequently visited districts
	// k int

	// number of districts 
	DistrictsAmount int

	// should we store which node this 
	// map is associated with
	Node int

	// total number of nodes 
	TotalNodes int
}

// initializes an empty graph
func NewGlobalMobilityGraph() *GlobalMobilityGraph {
	return &GlobalMobilityGraph{
		Map: make(map[Edge]float32),
		TotalNodes: 0,
	}
}

// generate the graph based on the private mobility graphs
func (g *GlobalMobilityGraph) GenerateGraph(privateMobilityGraphs map[model.NodeId]*PrivateMobilityGraph) {
	// iterate through each private mobility graph 
	for _, privateGraph := range privateMobilityGraphs {
		// iterate through each edge in each graph
		for edge := range privateGraph.Map {
			// add the edge to the global graph
			g.AddEdge(edge.District1, edge.District2)
		}
	}

	// set the total number of nodes
	g.TotalNodes = len(privateMobilityGraphs)

	// now that we are done adding all the edges of all graphs
	// we can normalize the graph 
	g.Normalize()
}

// adding DP noise to the global mobility graph 
func (g *GlobalMobilityGraph) Privatize(sigma float64) {
	// iterate through each edge in the graph
	for edge, weight := range g.Map {
		// add noise to the weight
		g.Map[edge] = weight + float32(sigma*rand.NormFloat64())
	}
}

// sybil flooding attack
// sets all weights to 0.99 
func (g *GlobalMobilityGraph) SybilFloodingAttack() {
	// iterate through each edge in the graph
	for edge := range g.Map {
		// set the weight to 0.99
		g.Map[edge] = 0.99
	}
}

// THIS SHOULD ONLY BE USED INTERNALLY 

// adds one edge to the graph
// if the edge exists, we increment the weight
// if the edge does not exist, we add it to the graph
func (g *GlobalMobilityGraph) AddEdge(district1, district2 int) {

	edge := Edge{district1, district2}
	if g.HasEdge(district1, district2) {
		g.Map[edge] += 1.0
	} else {
		g.Map[edge] = 1.0
	}
}

func (g *GlobalMobilityGraph) Normalize() {
	for edge, weight := range g.Map {
		g.Map[edge] = weight / float32(g.TotalNodes)
	}
}

func (g *GlobalMobilityGraph) GetEdgeWeight(district1, district2 int) float32 {
	edge := Edge{district1, district2}
	if weight, ok := g.Map[edge]; ok {
		return weight
	}
	return 0.0
}

// returns if the map has the edge based on districts 
func (g *GlobalMobilityGraph) HasEdge(district1, district2 int) bool {
	edge := Edge{district1, district2}
	if _, ok := g.Map[edge]; ok {
		return true
	}
	return false
}

