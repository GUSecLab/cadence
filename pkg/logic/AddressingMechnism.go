package logic

import (
	"fmt"
	model "marathon-sim/datamodel"
	"math"
	"sort"
	"strconv"
	"sync"
)

// Define a struct to hold your [3]int and float64
type AddressToScore struct {
	Address [3]int
	Score   float64
}

// this function checks if the node desire to accept
// a message, based on its threshold
func CheckThreshold(nodeid model.NodeId, geo Point3D) bool {
	//find the enclosing cube
	Address := EnclosingCubeMin(Profilier.Area.MinPoint, float32(Profilier.GridData.Distance), geo)
	//change here of the alternative does not work as well
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

func worker(tasks <-chan func(), wg *sync.WaitGroup) {
	defer wg.Done()
	for task := range tasks {
		task()
	}
}

func startWorkerPool(numWorkers int, tasks <-chan func()) *sync.WaitGroup {
	var wg2 sync.WaitGroup
	wg2.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(tasks, &wg2)
	}
	return &wg2
}

func Similarity(Address Point3D, id model.NodeId, wg *sync.WaitGroup, resultChan chan<- float64) {
	defer wg.Done()

	// Create a result channel with a buffer to accommodate all potential results.
	// result := make(chan float64)

	location_profile_value, _ := Profilier.LocationUpdates.LoadOrStore(id, 0)
	if location_profile_value == 0 {
		fmt.Printf("Node %v does not exist. Logic problem\n", id)
		resultChan <- 0
		return
	}

	node_data, ok := Profilier.Area.LookupMapLocation.Load(id)
	if !ok {
		fmt.Printf("Serious problem with Node %v, as it does not exist. Logic problem\n", id)
		resultChan <- 0
		return
	}
	sums := 0.0
	// var innerWg sync.WaitGroup
	node_data.(*sync.Map).Range(func(k, v interface{}) bool {
		// innerWg.Add(1)
		// go func(key, value interface{}) {
		// defer innerWg.Done()

		addressCube, ok := k.(Point3D)
		if !ok {
			// innerWg.Done()
			return true
		}
		score, ok := v.(int)
		if !ok {
			// innerWg.Done()
			return true
		}

		if addressCube == Address {
			sums += (float64(score) / float64(location_profile_value.(int)))
			// innerWg.Done()
			return true
		}

		real_score := float64(score) / float64(location_profile_value.(int))
		value_float := real_score / Distance3DPoint(addressCube, Address, Profilier.GridData.Gridsize)
		sums += value_float
		// innerWg.Done()
		// }(k, v)
		return true
	})

	// innerWg.Wait()
	// close(result)

	// var SumPaps float64
	// for val := range result {
	// 	SumPaps += val
	// }
	resultChan <- sums
}
func Similarity_General(Address Point3D, wg *sync.WaitGroup, resultChan chan<- float64) {
	defer wg.Done()
	simi, ok := Profilier.Area.SimilarityMapGeneral.Load(Address)
	if !ok || simi == nil {
		resultChan <- float64(0.0)
		return
	}
	resultChan <- simi.(float64)

}

func Similarity_General_Old(Address Point3D, wg *sync.WaitGroup, resultChan chan<- float64) {
	defer wg.Done()
	SumPaps := 0.0
	result := make(chan float64)
	taskChan := make(chan func())
	workerWg := startWorkerPool(5, taskChan) // Adjust the number of workers as needed

	Profilier.Area.LookupMapGeneral.Range(func(k, v interface{}) bool {
		taskChan <- func() {
			addressCube, ok := k.(Point3D)
			if !ok {
				return
			}

			score, ok := v.(float32)
			if !ok {
				return
			}

			if addressCube == Address {
				result <- float64(score)
				return
			}

			value := float64(score) / Distance3DPoint(addressCube, Address, Profilier.GridData.Gridsize)
			result <- value
		}
		return true
	})

	close(taskChan)
	workerWg.Wait()
	close(result)
	for val := range result {
		SumPaps += val
	}
	resultChan <- SumPaps
}

// this function computes the relationship between
// a node n's similarity and that of the general node's similarity
func MarginalSimilarity(nodeid model.NodeId, Address Point3D) float64 {
	var SimilarityBarrier sync.WaitGroup
	SimilarityBarrier.Add(2)
	simi_loc_channel := make(chan float64, 1)
	simi_gen_channel := make(chan float64, 1)
	go Similarity(Address, nodeid, &SimilarityBarrier, simi_loc_channel)
	go Similarity_General(Address, &SimilarityBarrier, simi_gen_channel)
	SimilarityBarrier.Wait()
	// Read the two values and close channels
	simi_loc := <-simi_loc_channel
	simi_gen := <-simi_gen_channel
	close(simi_loc_channel)
	close(simi_gen_channel)
	return simi_loc / simi_gen
}

// thresholds update - for threshold marginal similarity score
// at which a node accepts a message and becomes a carrier
func ThresholdUpdate(nodeid model.NodeId, barrier *sync.WaitGroup, threshChan chan *model.Thresholds, dataset string, participants int) {
	SigmasList := make([]float64, 0)
	//eliminate invalid nodes
	node_data, ok := Profilier.Area.LookupMapLocation.Load(nodeid)
	if !ok {
		logg.Infof("Serious problem with Node %v, as it does not exist. Logic problem", model.NodeIdString(nodeid))
		threshChan <- nil
		barrier.Done()
	}
	// Iterate over the sync.Map
	node_data.(*sync.Map).Range(func(k, v interface{}) bool {
		Address, ok := k.(Point3D)
		if !ok {
			fmt.Println("problem with getting adresses for threshold creation")
		}
		//calculate the little sigma array - change here to alternative
		//pay attention if want to go back
		LittleSigmaLine := MarginalSimilarity(nodeid, Address)
		//append the LittleSigmaLine to the list
		SigmasList = append(SigmasList, LittleSigmaLine)
		// Print key and value
		//fmt.Printf("Key: %v, Value: %v\n", k, v)
		// Continue iterating
		return true
	})

	//sort the sigmas
	sort.Float64s(SigmasList)
	//calculate i in reference to the threshold
	
	// k := participants
	k := 10
	lensig := len(SigmasList)
	ks := float64((k - 1) / k)
	i := int(math.Ceil(float64(lensig) * ks))

	//calculate tau
	tau := SigmasList[i]
	//update the threshold

	Profilier.NodesThresholds.Store(nodeid, tau)
	// save the threshold to the channel
	th := model.Thresholds{Node: nodeid, Threshold: tau, Dataset: dataset}
	fmt.Println("node ", nodeid, " is having threshold ", strconv.FormatFloat(tau, 'f', 6, 64))
	threshChan <- &th
	barrier.Done() //sign the this threshold was created
}

// this function checks if a node should acquire a message.
// it decises by the type of the message's destination
func CheckAcquire(m *Message, nodeid model.NodeId) bool {
	//checking the
	g := m.Destination.(Point3D)
	return CheckThreshold(nodeid, g)

}