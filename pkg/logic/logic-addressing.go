package logic

import (
	"database/sql"
	"encoding/json"
	"fmt"
	model "marathon-sim/datamodel"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type AddressingLogic struct {
	log *logger.Logger

	// a map from NodeId to NodeState
	statesMap sync.Map

	// a map from NodeId to a map of messagesMap
	messagesMap sync.Map
}

// popular locations for each node struct
// in this struct, we enlist the most popular
// and least popular places in the grid
type NodesPopular struct {
	MostPopular  map[int][3]float64
	LeastPopular map[int][3]float64
}

// nodes profiling struct
type NodesLocations struct {
	GeneralLocationProfile map[[3]int]float64 //general location profile
	GeneralUpdates         int                //general couner for movements
	LocationProfile        sync.Map           //location profile per node
	LocationUpdates        sync.Map           //number of updates per node
	GridData               GridInput          //grid data
	NodesThresholds        sync.Map           //thersholds for nodes
	NodesAmount            int
	Popularity             sync.Map //popularity map
	MinX                   float32  //minimum x value of addresses
	MinY                   float32  //minimum y value of addresses
	MinZ                   float32  //minimum y value of addresses
}

var Profilier NodesLocations

// struct for storing the data for grid creation
type GridInput struct {
	Gridsize float64 `json:"gridsize"`
	Distance float64 `json:"distance"`
}

// read profile files
func (al *AddressingLogic) SetProfileParams(profileFile string) {
	// load profile file
	dat, err := os.ReadFile(profileFile)
	if err != nil {
		al.log.Info("Problem with profile file reading")
	}

	// the format of the JSON file is an array of Message objects (see `Message`
	// above)
	var gridinput *GridInput
	if err = json.Unmarshal(dat, &gridinput); err != nil {
		al.log.Infof("Problem with grid input reading: %v", err)
	}
	//update the profilier grid data
	Profilier.GridData.Gridsize = gridinput.Gridsize
	Profilier.GridData.Distance = gridinput.Distance

}

// initlizing the logics with the content block and a log
// in the addressing case, set the grid parameters
// and init the popularities,locations and thresholds
func (al *AddressingLogic) InitLogic(block *LogicConf, log *logger.Logger) {
	//init logger
	al.log = log
	//set grid parameters
	al.SetProfileParams(block.ProfileFile)
	//init thresholds list
	Profilier.NodesThresholds = sync.Map{}
	//init the location grid
	Profilier.LocationUpdates = sync.Map{}
	//init popularity map
	Profilier.Popularity = sync.Map{}
}

// This is more of an example than a useful function.  Whenever a node updates
// its position and this function is called, NewPositionCallback will update the
// state variable `lastpos` with the current position.  This isn't really used
// anywhere, but is intended to demonstrate how state is handled.
func (al *AddressingLogic) NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) {
	//state change, modify in the future
	var stateMap *sync.Map
	state, ok := al.statesMap.Load(nodeid)
	if ok {
		stateMap = state.(*sync.Map)
	} else {
		stateMap = new(sync.Map)
		al.statesMap.Store(nodeid, stateMap)
	}
	//update the location of the node in the state map
	stateMap.Store("lastpos", b)

	//update the amount of locations, and then the profile of the node

	tmp_location_value, _ := Profilier.LocationUpdates.LoadOrStore(nodeid, 0)
	Profilier.LocationUpdates.Store(nodeid, tmp_location_value.(int)+1)

	//change the profile of the node
	pos, err := model.Unmarshall(b, t)
	if err != nil {
		al.log.Warnf("Cannot decipher location: %v", err)
	}
	cordinates := pos.ConvertToCoord()
	//split the dimensions
	tmp_ar := []float64{cordinates.X, cordinates.Y, cordinates.Z}
	//update the specific location for the nodeid
	up_nodeid, ok := Profilier.LocationProfile.Load(nodeid)
	if !ok {
		logg.Infof("error loading location profile of nodide %v, abort!", model.NodeIdString(nodeid))
		return
	}
	UpdateLocationProfile(tmp_ar, Profilier.GridData.Gridsize, up_nodeid.(map[[3]int]int), Profilier.GridData.Distance)
	//update the location profile
	Profilier.LocationProfile.Store(nodeid, up_nodeid)
}

// data preparation
func (al *AddressingLogic) DataPreparation(rows *sql.Rows, node_list *sql.Rows, logg *logger.Logger, minevents int) (sync.Map, []model.NodeId) {
	return LocationsCreation(rows, node_list, Profilier.GridData.Gridsize, Profilier.GridData.Distance, logg, minevents)
}

// create both general and profile locations
// a set of random 10% of the nodes will be used as general location profile
// the other 90% will be used for the location profile
func LocationsCreation(rows *sql.Rows, node_list *sql.Rows, gridSize float64, epsilon float64, logg *logger.Logger, minevents int) (sync.Map, []model.NodeId) {
	nodes_names := make(map[model.NodeId]bool)
	nodes_name_slice := make([]model.NodeId, 0)
	nodes_data := make(map[model.NodeId][]model.Event) //map of floats of the data
	Profilier.NodesAmount = 0
	minx := math.Inf(1)
	miny := math.Inf(1)
	minz := math.Inf(1)
	//Step 1: arrange the nodes
	for rows.Next() {
		var ev model.Event
		if err := rows.Scan(&ev.DatasetName, &ev.Time, &ev.Node, &ev.MarshalledLocation, &ev.X, &ev.Y, &ev.Z); err != nil {
			logg.Warnf("invalid event %v", err)
			continue
		}
		//update minimum values for later
		if ev.X < minx {
			minx = ev.X
		}
		if ev.Y < miny {
			minx = ev.Y
		}
		if ev.Z < minz {
			minz = ev.Z
		}
		//update nodeid name
		name := ev.Node
		if !nodes_names[name] {
			nodes_names[name] = true
			nodes_name_slice = append(nodes_name_slice, name)
		}
		nodes_data[name] = append(nodes_data[name], ev)
	}
	nodes_name_slice_new := make([]model.NodeId, 0)
	//delete sparse nodes
	for _, k := range nodes_name_slice {
		if len(nodes_data[k]) > minevents {
			nodes_name_slice_new = append(nodes_name_slice_new, k)
		}
	}
	//update the amount of nodes
	Profilier.NodesAmount = len(nodes_name_slice_new)
	nodes_name_slice = nodes_name_slice_new
	//update the profilier about minimum values
	Profilier.MinX = float32(minx)
	Profilier.MinX = float32(miny)
	Profilier.MinZ = float32(minz)
	//Step 2: randomize the nodes for sampling
	//the general location
	// Seed the random number generator with the current time

	// Generate a random permutation of the slice indices
	//follow randomness seed

	randIdx := model.Perm(Profilier.NodesAmount)

	// Create a new slice with the same elements in a random order
	rand_nodes_name_slice := make([]model.NodeId, len(nodes_name_slice))
	for i, idx := range randIdx {
		rand_nodes_name_slice[i] = nodes_name_slice[idx]
	}
	//!!erase when you want ranomity!!
	rand_nodes_name_slice = nodes_name_slice
	//Step 3: split the nodes to profile and general locations
	split_node := int(math.Ceil(0.9 * float64(Profilier.NodesAmount)))
	regularnodes := rand_nodes_name_slice[:split_node]
	generalnodes := rand_nodes_name_slice[split_node:]

	//Step 4: use each list to create the different locations
	nodes_data_general := make(map[model.NodeId][]model.Event) //map of floats of the data
	nodes_data_regular := make(map[model.NodeId][]model.Event) //map of floats of the data
	//list the data of the regular nodes
	for _, k := range regularnodes {
		nodes_data_regular[k] = nodes_data[k]
	}
	//list the data of the general nodes
	for _, k := range generalnodes {
		nodes_data_general[k] = nodes_data[k]
	}
	//Step 5: Iterate each list seperately
	//general profile
	var LocationsBarrier sync.WaitGroup //locations barrier
	LocationsBarrier.Add(2)             //add 2 for the go-routine
	go GeneralLocationProfileCreation(nodes_data_general, gridSize, epsilon, logg, &LocationsBarrier)

	//location profile and leftovers times for the rest of the function
	times_channel := make(chan sync.Map)
	go LocationProfileCreation(nodes_data_regular, gridSize, epsilon, logg, times_channel, &LocationsBarrier)
	times := <-times_channel
	LocationsBarrier.Wait() //wait for the locations to be created
	close(times_channel)    //close channel
	logg.Infof("profiles created! now we return only the relevant data")

	//Step 6: creating initial threshold
	logg.Infof("creating initial thresholds")
	// let's get a list of nodes
	var thresholdBarrier sync.WaitGroup //threshold barrier
	//update thresholds for starting the simulation
	thresholdBarrier.Add(len(regularnodes)) //add 1 for the go-routine
	for _, key := range regularnodes {
		logg.Infof("initilaizing threshold for node %v", model.NodeIdString(key))
		go ThresholdUpdate(key, &thresholdBarrier)
	}
	thresholdBarrier.Wait() //wait for the threshold to be created
	//convert the map of nodes->
	//return min/max times for each node

	return times, regularnodes
}

// this function finds the correct grid for a geolocation address
func LocateGridSquare(geo []float64, gridSize float64, counters map[[3]int]int, epsilon float64) [3]int {
	// Find the grid cell for the current geolocation
	gridPos := [3]int{int(geo[0] - float64(Profilier.MinX)/gridSize), int(geo[1] - float64(Profilier.MinX)/gridSize), int(geo[2] - float64(Profilier.MinX)/gridSize)}
	// Check if there is a nearby geolocation that was already counted in a grid cell
	//if there is, return it
	for pos := range counters {
		dist := Distance(pos, gridPos, gridSize)
		if dist < epsilon {
			return pos
		}
	}

	// If no nearby geolocation was found, return a new grid cell
	return gridPos

}

// update location profile
func UpdateLocationProfile(geo []float64, gridSize float64, counters map[[3]int]int, epsilon float64) {
	//get the address

	address := LocateGridSquare(geo, gridSize, counters, epsilon)

	_, exists := counters[address]
	if exists {
		counters[address]++
	} else {
		counters[address] = 1
	}

}

// create location profile, dump it to a dedicated channel at the end
func LocationProfileCreation(nodes map[model.NodeId][]model.Event, gridSize float64, epsilon float64, logg *logger.Logger, ch chan<- sync.Map, barrier *sync.WaitGroup) {
	logg.Infof("starting location profile creation")
	//create a leftover map for the actual data for the experiment
	times := sync.Map{}

	for key := range nodes {
		// Get the length of the slice
		originalLen := len(nodes[key])

		// Calculate the index up to which we want to keep the elements
		endIdx := int(0.1 * float64(originalLen))
		//sort the slice according to time, the last element
		sortedSlice := nodes[key]
		sort.Slice(sortedSlice, func(i, j int) bool {
			return sortedSlice[i].Time < sortedSlice[j].Time
		})
		// Use slicing to create a new slice with only the first 30% of the original elements
		newSlice := sortedSlice[:endIdx]
		//save the other data for the experiment
		left_items := sortedSlice[endIdx:]
		//save the first and last time of each node
		times.Store(key, left_items)
		// Replace the value of the current key with the new slice
		nodes[key] = newSlice
	}
	//create the barrier for the specific locations creation
	var loc_barrier sync.WaitGroup
	loc_barrier.Add(len(nodes))
	for id, geos_tmp := range nodes {
		geos := make([][]float64, 0)
		for _, geo := range geos_tmp {
			geos = append(geos, []float64{geo.X, geo.Y, geo.Z})
		}
		//create the profile location for a specific nodeid
		SpecificLocationProfileCreation(id, geos, gridSize, epsilon, logg, &loc_barrier)
	}
	loc_barrier.Wait()
	//return the data the was not used to the main process
	//the data will only incluce the nodeid and time (min/max)
	//so that the nodeworker will be able to create its
	//interval
	ch <- times
	//update barrier that we are done
	barrier.Done()
}

// geolocation creation for a specific node
func SpecificLocationProfileCreation(id model.NodeId, geos [][]float64, gridSize float64, epsilon float64, logg *logger.Logger, barrier *sync.WaitGroup) {
	logg.Infof("starting location profile of node %v", id)
	// Create a grid of counters for the geolocations
	tmp_grid := make(map[[3]int]int)

	for _, geo := range geos {
		//update the amount of update counted,globally/specifically
		tmp_location_value, _ := Profilier.LocationUpdates.LoadOrStore(id, 0)
		Profilier.LocationUpdates.Store(id, tmp_location_value.(int)+1)
		//update the location grif with the geolocation
		UpdateLocationProfile(geo, gridSize, tmp_grid, epsilon)

	}
	//create the popular places for the node
	//when creating messages, we will use it
	PopularCreation(tmp_grid, gridSize, epsilon, logg, id)
	//store the location profile of the nodeid
	//in the syncmap
	Profilier.LocationProfile.Store(id, tmp_grid)
	//signal the barrier that this location was created
	barrier.Done()
}

// create popular places for each node
func PopularCreation(node_data map[[3]int]int, gridSize float64, epsilon float64, logg *logger.Logger, id model.NodeId) {
	// create a slice of key-value pairs
	var keyValuePairs []struct {
		key   [3]int
		value int
	}
	//add the key,values from the grid of the node
	for k, v := range node_data {
		keyValuePairs = append(keyValuePairs, struct {
			key   [3]int
			value int
		}{k, v})
	}

	// sort the slice based on the value in descending order
	sort.SliceStable(keyValuePairs, func(i, j int) bool {
		return keyValuePairs[i].value > keyValuePairs[j].value
	})
	//create the struct for the popular places
	var pop NodesPopular
	pop.LeastPopular = make(map[int][3]float64)
	pop.MostPopular = make(map[int][3]float64)

	//update the most popular and least popular values
	for i := 0; i < 5 && i < len(keyValuePairs); i++ {
		pop.MostPopular[i] = [3]float64{(float64(keyValuePairs[i].key[0]) - float64(Profilier.MinX)) / gridSize, (float64(keyValuePairs[i].key[1]) - float64(Profilier.MinY)) / gridSize, (float64(keyValuePairs[i].key[2]) - float64(Profilier.MinZ)) / gridSize}
	}
	counter_least := 0
	for i := len(keyValuePairs) - 1; i >= len(keyValuePairs)-5 && i >= 0; i-- {
		pop.LeastPopular[counter_least] = [3]float64{(float64(keyValuePairs[i].key[0]) - float64(Profilier.MinX)) / gridSize, (float64(keyValuePairs[i].key[1]) - float64(Profilier.MinY)) / gridSize, (float64(keyValuePairs[i].key[2]) - float64(Profilier.MinZ)) / gridSize}
		counter_least++
	}

	//save the popularity addresses
	Profilier.Popularity.Store(id, pop)

}

// general location profile
func GeneralLocationProfileCreation(nodes map[model.NodeId][]model.Event, gridSize float64, epsilon float64, logg *logger.Logger, barrier *sync.WaitGroup) {
	logg.Infof("starting general profile creation")
	//create global counter and global movements map
	GlobalMovements := make(map[[3]int]float64)
	GlobalMovementsCount := 0
	for id, geos := range nodes {
		logg.Infof("starting general profile of node %v", id)
		// Create a grid of counters for the geolocations
		for _, geo_event := range geos {
			geo := [3]float64{geo_event.X, geo_event.Y, geo_event.Z}
			// update the amount of update counted,globally/specifically
			GlobalMovementsCount++
			// Find the grid cell for the current geolocation
			gridPos := [3]int{int(geo[0] - float64(Profilier.MinX)/gridSize), int(geo[1] - float64(Profilier.MinY)/gridSize), int(geo[2] - float64(Profilier.MinZ)/gridSize)}

			// Check if there is a nearby geolocation that was already counted in a grid cell
			found := false
			for pos := range GlobalMovements {
				dist := Distance(pos, gridPos, gridSize)
				if dist < epsilon {
					GlobalMovements[pos]++
					found = true
					break
				}
			}

			// If no nearby geolocation was found, create a new grid cell
			if !found {
				GlobalMovements[gridPos] = 1
			}

		}

	}
	//divide the movements per cell
	//by the amount of cells
	//to get the average of the population
	for key, value := range GlobalMovements {
		GlobalMovements[key] = value / float64(GlobalMovementsCount)
		fmt.Print(key, GlobalMovements[key])

	}

	//saving the globals and specific profiles
	Profilier.GeneralLocationProfile = GlobalMovements
	Profilier.GeneralUpdates = GlobalMovementsCount

	//update barrier
	barrier.Done()
}

// calculates the location profile
func LocationCalculation(nodeid model.NodeId, Address [3]int) float64 {
	//eliminate invalid values
	tmp_location_value, _ := Profilier.LocationUpdates.LoadOrStore(nodeid, 0)
	//this means that the node practically does not exist
	if tmp_location_value == 0 {
		logg.Infof("Node %v does not exist. Logic problem", model.NodeIdString(nodeid))
		return 0
	}

	//location profile calculation
	tmp_loc, ok := Profilier.LocationProfile.Load(nodeid)
	if !ok {
		logg.Infof("error in calculating location profile of nodeid %v", model.NodeIdString(nodeid))
		return 0
	}
	score := tmp_loc.(map[[3]int]int)[Address]
	return float64(score) / float64(tmp_location_value.(int))
}

func (al *AddressingLogic) HandleHelper(encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) float32 {
	actual_bandwidth := float32(0.0)
	// iterate thru messages on node1 and add them all to node2
	messageMap1.Range(func(k, v any) bool {

		messageid := k.(string)
		message := v.(*Message)

		//here we need to check if the receiver's threshold prohibits from
		//transefering the message or not
		//the check will be based on the message from nodeid1
		//and the id of nodeid2 (the potential recipient)
		if !CheckAcquire(message, nodeid2) {
			return true
		}
		// lock message transfering
		nodemem, _ := Storage.NodesMemories.Load(nodeid2)
		nodemem_serialized := nodemem.(NodeMemory)
		nodemem_mutex := nodemem_serialized.NodeMutex
		nodemem_mutex.Lock()
		//hand over the nandling to the general logic function,
		//as it works for a couple of logics
		transfer := TransferMessage(encounter, messageMap1, messageMap2, nodeid1, nodeid2, messageid, message)
		if transfer {
			//update transfer counter of messages
			actual_bandwidth++
		}
		//unlock message transferring
		nodemem_mutex.Unlock()
		return true
	})
	return actual_bandwidth
}

// transfers all messages between the nodes
func (al *AddressingLogic) HandleEncounter(encounter *model.Encounter) float32 {

	// get the two node IDs involved
	nodeid1 := encounter.Node1
	nodeid2 := encounter.Node2

	// get each node's message map
	messageMap1 := GetMessageQueue(nodeid1)
	messageMap2 := GetMessageQueue(nodeid2)

	band1 := al.HandleHelper(encounter, messageMap1, messageMap2, nodeid1, nodeid2)
	band2 := al.HandleHelper(encounter, messageMap2, messageMap1, nodeid2, nodeid1)
	//return aggregated bandwith
	return band1 + band2
}

// get the logic name
func (al *AddressingLogic) GetLogicName() string {
	return "addressing"
}

// loop over the geolocation array to create
// a string reporesentation.
// this is used for the least/most populated geolocations
func loop_address_for_detination(pops map[int][3]float64, m *Message, mpop bool, res map[model.NodeId][]*Message, startTime float64, id_count *int64) {

	for i, address := range pops {
		m_tmp_most := m.Copy() //tmp message for the populest addresses
		if mpop {              //popular geolocations, 5 is considered negative
			m_tmp_most.MPop = i
			m_tmp_most.LPop = 5
		} else {
			m_tmp_most.LPop = i
			m_tmp_most.MPop = 5
		}

		//concat it to the destination, as nodeid;geolocation
		//m_tmp_most.Destination = model.NodeIdString(node_des) + ";" + str.String()
		m_tmp_most.Destination = address
		//attach the id to the message
		m_tmp_most.MessageId = strconv.FormatInt(*id_count, 10)
		(*id_count)++
		//attach the shard id if the message is split
		if m_tmp_most.ShardsAvailable == 1 {
			m_tmp_most.MessageId = m_tmp_most.MessageId + "_" + strconv.Itoa(m_tmp_most.ShardID)
		}
		// create a new message accordingly
		res[m.Sender] = append(res[m.Sender], m_tmp_most)
	}

}
