package logic

import (
	model "marathon-sim/datamodel"
	"math"
	"os"
	"sort"
	"sync"
)

// this function checks if the node desire to accept
// a message, based on its threshold
func CheckThreshold(nodeid model.NodeId, geo []float64) bool {
	loc, ok := Profilier.LocationProfile.Load(nodeid)
	//error in location getter
	if !ok {
		logg.Infof("error in location access for threshold of nodeid %v, aborting threshold calc", model.NodeIdString(nodeid))
		return true
	}
	Address := LocateGridSquare(geo, Profilier.GridData.Gridsize, loc.(map[[3]int]int), Profilier.GridData.Distance)
	MarSim := MarginalSimilarity(nodeid, Address)
	//if the marginal similarity is at least the threshold,
	//return true.
	//elsewhere, return false.
	threshold, ok := Profilier.NodesThresholds.Load(nodeid)
	if !ok {
		logg.Infof("threshold loading problem for nodeid %v", nodeid)
		return false
	}

	return MarSim >= threshold.(float64)
}

// function that calculates the similarity of a node
// A node determines if it is the best of k − 1 other message
// recipients by comparing its similarity with
// the message’s destination to the "average" node's similarity
func Similarity(profile map[[3]int]int, Address [3]int, id model.NodeId) float64 {
	Pam := float64(profile[[3]int{Address[0], Address[1], Address[2]}])

	SumPaps := 0.0
	// iterate over the 3D int map
	for key := range profile {
		if key == Address { //ignore the case of Pam
			continue
		}
		SumPaps += LocationCalculation(id, key) / Distance(key, Address, Profilier.GridData.Gridsize)

	}
	return Pam + SumPaps
}

// function that calculates the similarity of a node
// A node determines if it is the best of k − 1 other message
// recipients by comparing its similarity with
// the message’s destination to the "average" node's similarity
func Similarity_General(profile map[[3]int]float64, Address [3]int) float64 {
	Pam := profile[[3]int{Address[0], Address[1], Address[2]}]

	SumPaps := 0.0
	// iterate over the 3D int map
	for key := range profile {
		if key == Address { //ignore the case of Pam
			continue
		}
		SumPaps += Profilier.GeneralLocationProfile[key] / Distance(key, Address, Profilier.GridData.Gridsize)

	}
	return Pam + SumPaps
}

// this function computes the relationship between
// a node n's similarity and that of the general node's similarity
func MarginalSimilarity(nodeid model.NodeId, Address [3]int) float64 {
	loc, ok := Profilier.LocationProfile.Load(nodeid)
	//error in location getter
	if !ok {
		logg.Infof("error in location access for marginal similarity of nodeid %v, aborting it", model.NodeIdString(nodeid))
		os.Exit(0)
	}
	simi_prof := Similarity(loc.(map[[3]int]int), Address, nodeid)
	simi_gen := Similarity_General(Profilier.GeneralLocationProfile, Address)
	return simi_prof / simi_gen
}

// thresholds update - for threshold marginal similarity score
// at which a node accepts a message and becomes a carrier
func ThresholdUpdate(nodeid model.NodeId, barrier *sync.WaitGroup) {
	SigmasList := make([]float64, 0)
	//create the 3d array of addresses
	for Address := range Profilier.GeneralLocationProfile {
		//calculate the little sigma line
		LittleSigmaLine := MarginalSimilarity(nodeid, Address)
		//append the LittleSigmaLine to the list
		SigmasList = append(SigmasList, LittleSigmaLine)
	}
	//sort the sigmas
	sort.Float64s(SigmasList)
	//calculate i in reference to the threshold
	k := 20
	lensig := len(SigmasList)
	ks := float64((k - 1) / k)
	i := int(math.Ceil(float64(lensig) * ks))

	//calculate tau
	tau := SigmasList[i]
	//update the threshold

	Profilier.NodesThresholds.Store(nodeid, tau)
	barrier.Done() //sign the this threshold was created
}

// this function checks if a node should acquire a message.
// it decises by the type of the message's destination
func CheckAcquire(m *Message, nodeid model.NodeId) bool {
	//checking the
	g := m.Destination.([3]float64)
	gs := []float64{g[0], g[1], g[2]}
	return CheckThreshold(nodeid, gs)

}
