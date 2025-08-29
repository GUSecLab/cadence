package logic

import (
	"database/sql"
	model "marathon-sim/datamodel"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	logger "github.com/sirupsen/logrus"
)

type MirageLogic struct {
	// logger
	log *logger.Logger

	// dataset name
	DatasetName string

	// parameters:

	// number of districts 
	DistrictsAmount int

	// probability of flipping a bit 
	p float32

	// k top frequently visited districts 
	k int 

	// max N value
	MaxN int

	// we do not want to store this locally, but in a database
	// associates a node with its private mobility graph 
	NodePMG map[model.NodeId]*PrivateMobilityGraph
	
	// a list of districts 
	Districts []District

	// associates a node with its most frequently visited district
	// since our messages our addressed to nodes but our logic wants 
	// addresses to districts, we need this map
	NodeMostFrequentDistrict map[model.NodeId]int

	// global mobility graph
	GlobalMobilityGraph *GlobalMobilityGraph

	// creating sync maps for the N values
	// each node has its own map 
	NValues map[model.NodeId]*NValueMap

	// stores all the nodes in this experiment 
	AllNodes []model.NodeId

	// kmeans iterations 
	KMeansIterations int

	// scale N values 
	NFactor int

	// DP noise sigma parameter 
	NoiseSigma float64

	// attack type 
	AttackType string
}



func (ml *MirageLogic) InitLogic(LogicConfig *LogicConf, log *logger.Logger) {
	ml.DatasetName = LogicConfig.Dataset
	ml.DistrictsAmount = LogicConfig.DistrictsAmount
	ml.p = LogicConfig.P
	ml.k = LogicConfig.K
	ml.KMeansIterations = LogicConfig.KMeansIterations
	ml.MaxN = LogicConfig.MaxN
	ml.NFactor = LogicConfig.NFactor
	ml.NoiseSigma = LogicConfig.NoiseSigma
	ml.AttackType = LogicConfig.AttackType

	// if attackType is not noise or sybil or none, we set it to none
	if ml.AttackType != "noise" && ml.AttackType != "sybil" && ml.AttackType != "none" {
		ml.AttackType = "none"
	}

	// ml.NodePMG = make(map[model.NodeId]*PrivateMobilityGraph)

	// initialize the logger
	ml.log = log

	// initialize the districts
	ml.Districts = make([]District, ml.DistrictsAmount)

	ml.NodeMostFrequentDistrict = make(map[model.NodeId]int)

	// initialize the global mobility graph
	ml.GlobalMobilityGraph = NewGlobalMobilityGraph()

	// initialize the N values
	ml.NValues = make(map[model.NodeId]*NValueMap)

	// initialize the node PMG
	ml.NodePMG = make(map[model.NodeId]*PrivateMobilityGraph)

}

// unused 
func (ml* MirageLogic) NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) {
}

// prepare our data!

// takes as input:
// all events for a dataset
// the logger
// min events for a node to be relevant
// spits out:
// a map of NodeId -> slice of all events for the node
// relevant nodes
func (ml *MirageLogic) DataPreparation(rows *sql.Rows, logger *logger.Logger, minevents int, config *model.Config) (*sync.Map, []model.NodeId){
	
	
	logger.Infof("Preparing data for dataset %s", ml.DatasetName)

	// create some preliminary data structures to store data 
	NodeEventsMap := make(map[model.NodeId][]model.Event)
	
	// process the rows 
	rowCount := 0

	for rows.Next() {
		// keep a counter for rows for debugging purposes 
		rowCount++
		// output every 100000 rows
		if rowCount % 100000 == 0 {
			logger.Infof("Processed %d rows", rowCount)
		}

		// create a new event 
		var event model.Event
		if err := rows.Scan(&event.Time, &event.Node, &event.X, &event.Y, &event.Z); err != nil {
			logger.Warnf("Event Error: %v", err)
			continue
		}

		node := event.Node
		// if the node associated with the current event is not in our
		// map, we add it to our map and initialize a slice 
		if _, ok := NodeEventsMap[node]; !ok {
			NodeEventsMap[node] = []model.Event{}
		}
		// append the event to the slice for the node
		NodeEventsMap[node] = append(NodeEventsMap[node], event)
	}

	// Sort the events for each node in NodeEventsMap
	for node, _ := range NodeEventsMap {
		sort.Slice(NodeEventsMap[node], func(i, j int) bool {
			return NodeEventsMap[node][i].Time < NodeEventsMap[node][j].Time
		})
	}

	// split the events by time, so we can have separate event sets for 
	// district generation, private mobility graph generation, and 
	// simulation

	logger.Infof("Splitting data by time")
	KMeansDataMap, PMGDataMap, FinalNodeEventsMap := SplitNodeData(NodeEventsMap, config.Simulation.StartTime, config.Simulation.EndTime)

	// generate the districts 
	logger.Infof("Generating districts")

	// check if we have districts in the database
	districts := GetDistrictsFromDB(logger, config.Simulation.DatasetName)
	if len(districts) > 0 {
		logger.Infof("Found %d districts in the database", len(districts))
		// use the districts from the database
		ml.Districts = districts
	} else {
		logger.Infof("No districts found in the database, generating new districts")
		// generate new districts
		ml.Districts = GenerateDistricts(KMeansDataMap, ml.DistrictsAmount, logger, ml.KMeansIterations)
		// store the districts in the database
		StoreDistrictsInDB(ml.Districts, config, logger)
	}
	// use K means clustering to generate districts 
	// ml.Districts = GenerateDistricts(KMeansDataMap, ml.DistrictsAmount, logger, ml.KMeansIterations)

	// generate the private mobility graph for each node in parrallel 
	// also generates mapping of node to most frequently visited district 
	logger.Infof("Generating private mobility graph")
	logger.Infof("Generating mapping of node to most frequently visited district")

	// create a map of nodes to their private mobility graphs 
	// store the non private mobility graphs 
	NodeMG := make(map[model.NodeId]*PrivateMobilityGraph)
	ml.NodePMG, NodeMG = GeneratePrivateMobilityGraphs(PMGDataMap, ml)
	
	// NodePMG := GeneratePrivateMobilityGraphs(PMGDataMap, ml)

	// use our private mobility graphs to generate the global mobility graph
	logger.Infof("Generating global mobility graph")
	// ml.GlobalMobilityGraph.GenerateGraph(NodePMG)
	ml.GlobalMobilityGraph.GenerateGraph(NodeMG)

	// add an attack as necessary 
	switch ml.AttackType {
		case "sybil":
			logger.Infof("Applying sybil flooding attack to global mobility graph")
			// apply the sybil flooding attack to the global mobility graph
			ml.GlobalMobilityGraph.SybilFloodingAttack()
		case "noise":
			logger.Infof("Applying noise to global mobility graph with sigma %f", ml.NoiseSigma)
			// apply DP noise to the global mobility graph
			// add noise to the graph
			ml.GlobalMobilityGraph.Privatize(ml.NoiseSigma)
	}

	// add our PMGs to the database
	// StorePMGsInDB(NodePMG, config, logger)

	/*
	// iterate over GlobalMobilityGraph and print the edges and weights
	logger.Infof("Global Mobility Graph Edges:")
	for edge, weight := range ml.GlobalMobilityGraph.Map {
		/logger.Infof("Edge: %v, Weight: %f", edge, weight)
	}
	*/

	// create a slice to store relevant nodes
	relevantNodes := GetNodeIdsFromMap(NodeEventsMap)
	ml.AllNodes = relevantNodes;

	FinalNodeEventsSyncMap := sync.Map{}
	// populate the sync map with the final node events
	for nodeId, events := range FinalNodeEventsMap {
		FinalNodeEventsSyncMap.Store(nodeId, events)
	}

	// testing 
	logger.Infof("finished data preparation")
	// ml.TestDBConversions()

	return &FinalNodeEventsSyncMap, relevantNodes
}

func StoreDistrictsInDB(Districts []District, config *model.Config, logger *logger.Logger) {
	// database can be accessed using model.DB
	for _, district := range Districts {
		// store the district in the database
		districtRecord := model.District{
			DatasetName: config.Simulation.DatasetName,
			DistrictId: district.DistrictId,
			X: district.X,
			Y: district.Y,
			Z: district.Z,
		}
		if r := model.DB.Create(districtRecord); r.Error != nil {
			logger.Infof("Cannot save district %d", district.DistrictId)
		}
	}
}

func GetDistrictsFromDB(logger *logger.Logger, datasetName string) []District {
	// create a map to store the districts
	districts := make([]District, 0)

	rows, err := model.DB.Table("districts").Where("dataset_name = ?", datasetName).Rows()	
	if err != nil {
		logger.Infof("Error getting districts from DB: %v", err)
		return districts
	}
	defer rows.Close()

	for rows.Next() {
		var district model.District
		if err := rows.Scan(&district.DatasetName, &district.DistrictId, &district.X, &district.Y, &district.Z); err != nil {
			logger.Infof("Error scanning district: %v", err)
			continue
		}

		// Add the district to the slice
		districts = append(districts, District{
			DistrictId: district.DistrictId,
			X: district.X,
			Y: district.Y,
			Z: district.Z,
		})
	}

	return districts
}


func StorePMGsInDB(NodePMG map[model.NodeId]*PrivateMobilityGraph, config *model.Config, logger *logger.Logger) {
	// database can be accessed using model.DB
	for nodeId, pmg := range NodePMG {
		// iterate over all edges in the PMG 
		for edge, _ := range pmg.Map {
			// store the edge in the database
			edgeRecord := model.PMGEdge{
				ExperimentName: config.Simulation.ExperimentName,
				NodeId: nodeId,
				District1: edge.District1,
				District2: edge.District2,
			}
			if r := model.DB.Create(edgeRecord); r.Error != nil {
				logg.Infof("Cannot save edge (%d, %d) for node %d", edge.District1, edge.District2, nodeId)
			}
		}
	}
}

// test getPMG 
// returns the private mobility graph of all nodes 
func (ml *MirageLogic) TestDBConversions() {
	// iterate over all nodesIDs
	for _, nodeId := range ml.AllNodes {
		// get the PMG from the database
		PMG := GetPMGFromDB(nodeId, ml.log, ml)
		ml.log.Infof("PMG for Node %d: %v", nodeId, PMG)

		// get the most frequent district 
		mostFrequentDistrict := ml.NodeMostFrequentDistrict[nodeId]
		ml.log.Infof("Most Frequent District for Node %d: %d", nodeId, mostFrequentDistrict)
	}
}

func GetPMGFromDB(NodeId model.NodeId, logger *logger.Logger, ml *MirageLogic) *PrivateMobilityGraph {
	// create a map to store the PMGs
	PMG := NewPrivateMobilityGraph(ml.DistrictsAmount)
	rows, err := model.DB.Table("pmg_edges").Where("node_id = ?", NodeId).Rows()
	if err != nil {
		logger.Infof("Error getting PMG from DB: %v", err)
		return PMG
	}
    for rows.Next() {
        var experimentName string
        var nodeId int
        var district1, district2 int

        // Scan the row into individual fields
        if err := rows.Scan(&experimentName, &nodeId, &district1, &district2); err != nil {
            logger.Infof("Error scanning PMG edge: %v", err)
            continue
        }

        // Add the edge to the PMG
        PMG.AddEdge(district1, district2)
    }
	// close the rows
	defer rows.Close()
	return PMG
}

// helper function to get the node IDs from the map
func GetNodeIdsFromMap(nodeEventsMap map[model.NodeId][]model.Event) []model.NodeId {
    // Create a slice to store the node IDs
    nodeIds := make([]model.NodeId, 0, len(nodeEventsMap))

    // Iterate over the map and collect the keys (node IDs)
    for nodeId := range nodeEventsMap {
        nodeIds = append(nodeIds, nodeId)
    }

    return nodeIds
}

func GeneratePrivateMobilityGraphs(nodeEvents map[model.NodeId][]model.Event, ml *MirageLogic) (map[model.NodeId]*PrivateMobilityGraph, map[model.NodeId]*PrivateMobilityGraph) {

	// use a waitgroup for parrallelism 
	var wg sync.WaitGroup

	// mutex to safely access shared NodePMG map 
	var mu sync.Mutex

	// create a map to store the private mobility graphs
	NodePMG := make(map[model.NodeId]*PrivateMobilityGraph)
	NodeMG := make(map[model.NodeId]*PrivateMobilityGraph)

	// iterate over all nodes 
	for nodeId, events := range nodeEvents {
		wg.Add(1)

		// launch a function for each node 
		go func(nodeId model.NodeId, events []model.Event) {
			defer wg.Done() // decrement waitgroup counter when done 
			// generate the private mobility graph for this node
			privateMobilityGraph, nonPrivateMobilityGraph, NodeMostFrequentDistricts := GeneratePrivateMobilityGraph(events, ml.k, ml)
			// lock the mutex before accessing the shared map
			mu.Lock()
			NodePMG[nodeId] = privateMobilityGraph
			NodeMG[nodeId] = nonPrivateMobilityGraph

			// we use the most frequently visited district data to populate our map
			ml.NodeMostFrequentDistrict[nodeId] = NodeMostFrequentDistricts[0]
			mu.Unlock() // unlock the mutex
		}(nodeId, events)
	}

	// wait for all goroutines to finish
	wg.Wait()

	// convert all the graphs to bit vectors and store in the database

	// iterate over all graphs and convert to bit vectors and print
	/*
	for nodeId, privateMobilityGraph := range NodePMG {
		ml.log.Infof("PMG for Node original %d: %v", nodeId, privateMobilityGraph)
	}
	*/
	
	return NodePMG, NodeMG
}

// returns the private graph and a slice of most frequent districts, sorted, which is useful!
func GeneratePrivateMobilityGraph(Events []model.Event, k int, ml *MirageLogic) (*PrivateMobilityGraph, *PrivateMobilityGraph, []int) {

	// initialize an empty private graph 
	privateGraph := NewPrivateMobilityGraph(ml.DistrictsAmount)
	nonPrivateMobilityGraph := NewPrivateMobilityGraph(ml.DistrictsAmount)

	// map to store the frequency of each district - histogram 
	districtFrequency := make(map[int]int)

	// iterate through the events and count the frequency of each district
	for _, event := range Events {
		districtId := ml.MapEventToDistrict(event)
		districtFrequency[districtId]++
	}

	// Create a slice to store districts and their frequencies
	type districtCount struct {
		DistrictID int 
		Count 	   int
	}

	districtCounts := make([]districtCount, 0, len(districtFrequency))
	for districtId, count := range districtFrequency {
		districtCounts = append(districtCounts, districtCount{
			DistrictID: districtId,
			Count:      count,
		})
	}
	// Sort the districts by frequency in descending order
	sort.Slice(districtCounts, func(i, j int) bool {
		return districtCounts[i].Count > districtCounts[j].Count
	})

	// convert districtCounts to a slice of district IDs
	topDistricts := make([]int, 0, k)
	for i := 0; i < k && i < len(districtCounts); i++ {
		topDistricts = append(topDistricts, districtCounts[i].DistrictID)
	}
	// add the top k districts to the private graph
	privateGraph.AddKDistricts(topDistricts)
	nonPrivateMobilityGraph.AddKDistricts(topDistricts)

	// finally, private the graph!
	privateGraph.Privatize(ml.p)

	// generate a map for this node 
	return privateGraph, nonPrivateMobilityGraph, topDistricts
}

func GenerateDistricts(nodeEvents map[model.NodeId][]model.Event, districtsAmount int, logger *logger.Logger, iterations int) []District {
	// convert events into a single slice 
	var allEvents []model.Event 
	for _, events := range nodeEvents {
		allEvents = append(allEvents, events...)
	}

	logger.Infof("sorting events")

	// sort the events by time
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Time < allEvents[j].Time
	})

	// initialize centroids randomly 
	logger.Infof("initializing centroids")
	// this is a little sketchy, we use an event as a centroid
	// for convenience 
	centroids := make([]model.Event, districtsAmount)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < districtsAmount; i++ {
		centroids[i] = allEvents[rand.Intn(len(allEvents))]
	}

	// create a map to store which event belongs to which centroid cluster 
	clusterAssignments := make(map[int][]model.Event)

	// perform K-Means Clustering!!

	// each iteration takes a ton of time ...
	totalIteration := iterations
	for iteration := 0; iteration < totalIteration; iteration++ {	
		// clear previous cluster assignments 
		logger.Infof("Iteration: %d", iteration)
		for i := 0; i < districtsAmount; i++ {
			clusterAssignments[i] = []model.Event{}
		}
		// assign each event to the nearest centroid 
		for _, event := range allEvents {
			closestCentroid := 0
			minDistance := math.MaxFloat64
			for i, centroid := range centroids {
				distance := calculateDistance(event, centroid)
				if distance < minDistance {
					minDistance = distance
					closestCentroid = i
				}
			}
			clusterAssignments[closestCentroid] = append(clusterAssignments[closestCentroid], event)
		}

		// update centroids based on the mean of the assigned events
		for i := 0; i < districtsAmount; i++ {
			if len(clusterAssignments[i]) > 0 {
				centroids[i] = calculateCentroid(clusterAssignments[i])
			}
		}
	}

	// create districts slice with unique IDs and centroids 
	districts := make([]District, districtsAmount)
	for i := 0; i < districtsAmount; i++ {
		districts[i] = District{
			DistrictId: i,
			X: centroids[i].X,
			Y: centroids[i].Y,
			Z: centroids[i].Z,
		}
	}
	// return the districts
	return districts
}

// MapEventToDistrict maps an event to the closest district based on the centroid.
func (ml *MirageLogic) MapEventToDistrict(event model.Event) int {
    closestDistrictID := -1
    minDistance := math.MaxFloat64

    // Iterate through all districts to find the closest centroid
    for _, district := range ml.Districts {
        // Calculate the distance between the event and the district's centroid
        distance := calculateDistance(event, model.Event{
            X: district.X,
            Y: district.Y,
            Z: district.Z,
        })

        // Update the closest district if this one is closer
        if distance < minDistance {
            minDistance = distance
            closestDistrictID = district.DistrictId
        }
    }

    return closestDistrictID
}

// calculateDistance calculates the Euclidean distance between two events
func calculateDistance(event1, event2 model.Event) float64 {
    return math.Sqrt(math.Pow(float64(event1.X-event2.X), 2) +
        math.Pow(float64(event1.Y-event2.Y), 2) +
        math.Pow(float64(event1.Z-event2.Z), 2))
}

// calculateCentroid calculates the centroid of a cluster of events
func calculateCentroid(events []model.Event) model.Event {
    var sumX, sumY, sumZ float64
    for _, event := range events {
        sumX += float64(event.X)
        sumY += float64(event.Y)
        sumZ += float64(event.Z)
    }
    count := float64(len(events))
    return model.Event{
        X: float64(sumX / count),
        Y: float64(sumY / count),
        Z: float64(sumZ / count),
    }
}

// takes in a slice of general nodes and events and spits out 
// splits for first 10%, second 10%, and rest
// the passed in map is SORTED by time
func SplitNodeData(nodeEvents map[model.NodeId][]model.Event, startTime, endTime float32) (map[model.NodeId][]model.Event, map[model.NodeId][]model.Event, map[model.NodeId][]model.Event) {
	
	first10PercentMap := make(map[model.NodeId][]model.Event)
	second10PercentMap := make(map[model.NodeId][]model.Event)
	restMap := make(map[model.NodeId][]model.Event)
	// calculate the end time interval for the first 10% of data
	first10PercentEndTime := float64(startTime + (endTime-startTime)*0.10)
	// calculate the end time interval for the next 10% of data
	second10PercentEndTime := float64(startTime + (endTime-startTime)*0.20)

	// iterate through the node events map
	for nodeId, events := range nodeEvents {
		// iterate through the events for each node
		for _, event := range events {
			// check if the event is in the first 10% of data
			if event.Time <= first10PercentEndTime {
				first10PercentMap[nodeId] = append(first10PercentMap[nodeId], event)
			} else if event.Time <= second10PercentEndTime {
				// check if the event is in the second 10% of data
				second10PercentMap[nodeId] = append(second10PercentMap[nodeId], event)
			} else {
				// otherwise, it is in the rest of the data
				restMap[nodeId] = append(restMap[nodeId], event)
			}
		}
	}

	return first10PercentMap, second10PercentMap, restMap
}

// MapCoordToDistrict maps a coordinate (x, y, z) to the closest district based on the centroid.
func (ml *MirageLogic) MapCoordToDistrict(x, y, z float64) int {
    closestDistrictID := -1
    minDistance := math.MaxFloat64

    // Iterate through all districts to find the closest centroid
    for _, district := range ml.Districts {
        // Calculate the distance between the coordinate and the district's centroid
        distance := math.Sqrt(
            math.Pow(x-district.X, 2) +
                math.Pow(y-district.Y, 2) +
                math.Pow(z-district.Z, 2),
        )

        // Update the closest district if this one is closer
        if distance < minDistance {
            minDistance = distance
            closestDistrictID = district.DistrictId
        }
    }

    return closestDistrictID
}

func RoutingDecision(ReceiverPrivateMobilityGraph *PrivateMobilityGraph, currentDistrict, destinationDistrict int) bool {

	// if an edge exists between the two districts, return true
	if ReceiverPrivateMobilityGraph.HasEdge(currentDistrict, destinationDistrict) {
		return true
	}
	return false
}

func (ml *MirageLogic) CalculateN(currentDistrict, destinationDistrict int, NFactor int) int {
	// get the edge weight from the global mobility graph
	edgeWeight := ml.GlobalMobilityGraph.GetEdgeWeight(currentDistrict, destinationDistrict)

	if edgeWeight == 0.0 {
		edgeWeight = 0.1
	}

	N := int(math.Ceil(float64((ml.p * edgeWeight + (1.0 - ml.p) * (1.0 - edgeWeight))/(ml.p * edgeWeight))))

	N = N * NFactor

    return int(math.Min(float64(N), float64(ml.MaxN)))
}

// potentially does a message exchange between nodes when an encounter
// occurs.
func (ml *MirageLogic) HandleEncounter(config *model.Config, encounter *model.Encounter) (float32, float32) {

	// ml.log.Infof("Handling encounter between %d and %d", encounter.Node1, encounter.Node2)

	// get the two node IDs involved
	nodeid1 := encounter.Node1
	nodeid2 := encounter.Node2

	/*

	// lock message transfering
	nodemem_erase, _ := Storage.NodesMemories.Load(nodeid1)
	nodemem_serialized_erase := nodemem_erase.(*SimpleBuffer)
	nodemem_mutex_erase := nodemem_serialized_erase.NodeMutex
	nodemem_mutex_erase.Lock()
	nodemem, _ := Storage.NodesMemories.Load(nodeid2)
	nodemem_serialized := nodemem.(*SimpleBuffer)
	nodemem_mutex := nodemem_serialized.NodeMutex
	nodemem_mutex.Lock()

	*/

	messageMap1 := GetMessageQueue(nodeid1)
	messageMap2 := GetMessageQueue(nodeid2)

	//get ideal bandwidth and actual bandwidth
	band1, drops1 := ml.HandleHelper(config, encounter, messageMap1, messageMap2, nodeid1, nodeid2)
	band2, drops2 := ml.HandleHelper(config, encounter, messageMap2, messageMap1, nodeid2, nodeid1)

	return band1 + band2, drops1 + drops2
}

// handlehelper 
func (ml *MirageLogic) HandleHelper(config *model.Config, encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) (float32, float32) {

	// attempt to route the message

	// get the current district 
	currentDistrict := ml.MapCoordToDistrict(float64(encounter.X), float64(encounter.Y), float64(encounter.Z))

	actualBandwidth := 0.0
	droppedMessages := 0.0
	
	// iterate through the messages on node1 and attempt to pass each to node2
	messageMap1.Range(func(key, value interface{}) bool {
		// get the messageID	
		messageId := key.(string)
		// get the message	
		message := value.(*Message)
		// get the destination of the message
		destinationDistrict := ml.NodeMostFrequentDistrict[message.DestinationNode]

		
		
		// get the N value 
		NValueMap := ml.NValues[nodeid1]
		if NValueMap == nil {
			// create a new N value map for this node
			NValueMap = NewNValueMap()
			ml.NValues[nodeid1] = NValueMap
		}
		// get the N value for this message
		N, ok := NValueMap.Get(messageId, currentDistrict)
		if !ok {
			// the N value does not exist, so calculate it and put it in
			N = ml.CalculateN(currentDistrict, destinationDistrict, ml.NFactor)
		}

		// ml.log.Infof("Node %d is attempting to send message %s from district %v to district %d, will try %d times", nodeid1, messageId, currentDistrict, destinationDistrict, N)

		// check if we should attempt this transfer or not
		if N <= 0 {
			// skip this message
			return true
		}
		ml.NValues[nodeid1] = NValueMap

		// if we are here, N is positive!!

		// run the decision function for this message, currentDist, destDist set
		
		// get the private mobility graph for the receiver node
		// ReceiverPrivateMobilityGraph := GetPMGFromDB(nodeid2, ml.log, ml)
		ReceiverPrivateMobilityGraph := ml.NodePMG[nodeid2]
		if ReceiverPrivateMobilityGraph == nil {
			ml.log.Infof("Receiver Private Mobility Graph is nil for node %d", nodeid2)
			ReceiverPrivateMobilityGraph = NewPrivateMobilityGraph(ml.DistrictsAmount)
			// privatize the new graph 
			ReceiverPrivateMobilityGraph.Privatize(ml.p)
			// add the graph to the map
			ml.NodePMG[nodeid2] = ReceiverPrivateMobilityGraph
		}

		decision := RoutingDecision(ReceiverPrivateMobilityGraph, currentDistrict, destinationDistrict)

		if (decision == false) {
			// skip this mesage 
			return true
		}

		// the decision is true! we can transfer the message!
		// ml.log.Infof("ACCEPT! -> Node %d is transferring message %s to Node %d", nodeid1, messageId, nodeid2)
		transfer, didDrop := TransferMessage(config, encounter, messageMap1, messageMap2, nodeid1, nodeid2, messageId, message)
		if transfer {
			actualBandwidth++
		}

		if didDrop {
			// the message was dropped, increment dropped counter 
			droppedMessages++
		}

		// ml.log.Infof("transfer success!")

		// decrement the number of attempts
		N--

		
		// put N back in the map here!
		NValueMap.Set(messageId, currentDistrict, N)

		// put map back in the N value map
		ml.NValues[nodeid1] = NValueMap
		
		return true
	})
	
	// return the bandwidth
	return float32(actualBandwidth), float32(droppedMessages)
}

func (ml *MirageLogic) GetLogicName() string {
	return "mirage"
}