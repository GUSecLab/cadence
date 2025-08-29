package logic

import (
	"database/sql"
	"encoding/json"
	model "marathon-sim/datamodel"
	"math"
	"os"
	"sort"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type AddressingLogic struct {
	log *logger.Logger

	// a map from NodeId to NodeState
	statesMap sync.Map
	//store the dataset name for optimizing threshold
	//generation
	dataset string
}

// Worker represents a worker that processes a job
type Worker struct {
	ID             int
	JobChan        <-chan model.NodeId
	WG             *sync.WaitGroup
	ThresholdB     *sync.WaitGroup
	ThresholdSaver chan *model.Thresholds
}

func (w *Worker) Start(log *logger.Logger, dataset string, participants int) {
	defer w.WG.Done()

	for job := range w.JobChan {
		// Process the job (replace this with your actual processing logic)
		log.Infof("Worker %v processing node: %v\n", w.ID, job)
		ThresholdUpdate(job, w.ThresholdB, w.ThresholdSaver, dataset, participants)
	}
}

// nodes profiling struct
type NodesLocations struct {
	LocationUpdates sync.Map  //number of updates per node
	GridData        GridInput //grid data
	NodesThresholds sync.Map  //thersholds for nodes
	NodesAmount     int
	Area            Cube     //map area based on dataset's events
	Popularity      sync.Map //populatiry map for destiations of messages
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
	//store the dataset
	al.dataset = block.Dataset
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
	al.log.Infof("Bytes: %v", b)
	pos, err := model.Unmarshall(b, t)
	if err != nil {
		al.log.Warnf("Cannot decipher location: %v", err)
	}
	al.log.Infof("Position: %v", pos)
	cordinates := pos.ConvertToCoord()
	//update the cell in the cube
	cell := Point3D{float32(cordinates.X), float32(cordinates.Y), float32(cordinates.Z)}
	//update the movement for the node
	UpdateLocationProfile(cell, nodeid)

}

// data preparation
func (al *AddressingLogic) DataPreparation(rows *sql.Rows, logg *logger.Logger, minevents int, config *model.Config) (*sync.Map, []model.NodeId) {
	//generate times and nodes relevant for simulation
	times, nodes := LocationsCreation(rows, logg, minevents)

	//generate thresholds
	ThreshsGeneration(nodes, config.Simulation.DatasetName, logg)
	return &times, nodes

}

// create both general and profile locations
// a set of random 10% of the nodes will be used as general location profile
// the other 90% will be used for the location profile
func LocationsCreation(rows *sql.Rows, logg *logger.Logger, minevents int) (sync.Map, []model.NodeId) {

	gridSize := Profilier.GridData.Gridsize
	epsilon := Profilier.GridData.Distance
	nodes_names := make(map[model.NodeId]bool)
	nodes_name_slice := make([]model.NodeId, 0)
	nodes_data := make(map[model.NodeId][]model.Event) //map of floats of the data
	Profilier.NodesAmount = 0
	minx := math.Inf(1)
	miny := math.Inf(1)
	minz := math.Inf(1)
	maxx := math.Inf(-1)
	maxy := math.Inf(-1)
	maxz := math.Inf(-1)
	//Step 1: arrange the nodes
	for rows.Next() {
		var ev model.Event
		if err := rows.Scan(&ev.Time, &ev.Node, &ev.X, &ev.Y, &ev.Z); err != nil {
			logg.Infof("invalid event %v", err)
			continue
		}

		//update minimum and maximum values for later
		if ev.X < minx {
			minx = ev.X
		}
		if ev.X > maxx {
			maxx = ev.X
		}
		if ev.Y < miny {
			miny = ev.Y
		}
		if ev.Y > maxy {
			maxy = ev.Y
		}
		if ev.Z < minz {
			minz = ev.Z
		}
		if ev.Z > maxz {
			maxz = ev.Z
		}
		//update nodeid name
		name := ev.Node

		if _, ok := nodes_names[name]; !ok {
			nodes_names[name] = true
			nodes_name_slice = append(nodes_name_slice, name)
			nodes_data[name] = []model.Event{ev}
			continue
		}

		nodes_data[name] = append(nodes_data[name], ev)

	}
	// Sort nodes_data[name] by the Time field (assuming Time is a float64)
	for _, name := range nodes_name_slice {
		sort.Slice(nodes_data[name], func(i, j int) bool {
			return nodes_data[name][i].Time < nodes_data[name][j].Time
		})
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
	Profilier.Area.MinPoint = Point3D{float32(minx), float32(miny), float32(minz)}
	Profilier.Area.MaxPoint = Point3D{float32(maxx), float32(maxy), float32(maxz)}
	Profilier.Area.LookupMapGeneral = &sync.Map{}
	Profilier.Area.LookupMapLocation = &sync.Map{}
	Profilier.Area.SimilarityMapGeneral = &sync.Map{}

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
	//Step 4.5: generate the cube

	//Step 5: Iterate each list seperately
	//general profile
	var LocationsBarrier sync.WaitGroup //locations barrier
	LocationsBarrier.Add(1)             //add 2 for the go-routine
	go GeneralLocationProfileCreation(nodes_data_general, gridSize, epsilon, logg, &LocationsBarrier)

	//location profile and leftovers times for the rest of the function
	times := LocationProfileCreation(nodes_data_regular, gridSize, epsilon, logg)
	LocationsBarrier.Wait() //wait for the locations to be created
	//hack general locations - erase if you want to go back
	//HackGeneralLocation()
	logg.Infof("profiles created! now we return only the relevant data")

	return *times, regularnodes

}

func ThreshsGeneration(regularnodes []model.NodeId, dataset string, logg *logger.Logger) {
	// Step 6: creating initial threshold
	logg.Infof("creating initial thresholds")
	// let's get a list of nodes
	var thresholdBarrier sync.WaitGroup //threshold barrier
	// update thresholds for starting the simulation
	thresholdBarrier.Add(len(regularnodes) + 1) //add 1 for the go-routines
	//parllelism part
	// Set the number of workers and job queue size
	numWorkers := 30
	// Create a job channel
	jobChan := make(chan model.NodeId)
	// create a channel for documenting new thresholds
	thresholdChan := make(chan *model.Thresholds)
	//start a go routine that will document the thresholds
	go model.RecordThresholds(thresholdChan, &thresholdBarrier)
	// Create a WaitGroup to wait for all workers to finish
	var wg sync.WaitGroup
	//get the amount of participating nodes
	participants := len(regularnodes)
	// Create and start the worker pool
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		worker := Worker{ID: i, JobChan: jobChan, WG: &wg, ThresholdB: &thresholdBarrier, ThresholdSaver: thresholdChan}
		go worker.Start(logg, dataset, participants)
	}

	for _, key := range regularnodes {
		found := false // Flag to check if at least one row is found
		//check if the threshold already exists,
		//if so, skip the initilization
		// Create a variable to hold the result
		rows, err := model.DB.Table("thresholds").Select("threshold").Where("dataset=? and node=?", dataset, key).Rows()
		if err != nil {
			logg.Infof("error in fetching the threshold")
		} else {
			//get the threshold
			for rows.Next() {
				var cur_threshold float64
				if err := rows.Scan(&cur_threshold); err != nil {
					logg.Infof("error: %v", err)
					break
				}
				Profilier.NodesThresholds.Store(key, cur_threshold)
				found = true            //signal that a threshold was found
				thresholdBarrier.Done() //signal that this gorutine is done
				break
			}

		}
		// Check if any rows were found
		if !found {
			logg.Infof("no threshold found node %v", key)
			// Handle the case where no rows are found, e.g., return or log it
			logg.Infof("initilaizing threshold for node %v", model.NodeIdString(key))
			jobChan <- key
			//go ThresholdUpdate(key, &thresholdBarrier)
		}
		rows.Close()
	}
	// Close the job channel to signal that no more jobs will be added
	close(jobChan)

	// Wait for all workers to finish
	wg.Wait()
	//close channel documnetion
	close(thresholdChan)

	thresholdBarrier.Wait() //wait for the threshold to be created
}

func (al *AddressingLogic) HandleHelper(config *model.Config, encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) (float32, float32) {
	actual_bandwidth := float32(0.0)
	droppedMessages := float32(0.0)
	tmp_sync := *messageMap1
	// iterate thru messages on node1 and add them all to node2
	tmp_sync.Range(func(k, v any) bool {

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
		// nodemem, _ := Storage.NodesMemories.Load(nodeid2)
		// nodemem_serialized := nodemem.(*NodeMemory)
		// nodemem_mutex := nodemem_serialized.NodeMutex
		// nodemem_mutex.Lock()
		//hand over the nandling to the general logic function,
		//as it works for a couple of logics
		transfer, didDrop := TransferMessage(config, encounter, messageMap1, messageMap2, nodeid1, nodeid2, messageid, message)
		if transfer {
			//update transfer counter of messages
			actual_bandwidth++
		}

		// if the message was dropped, we need to update the memory
		if didDrop {
			droppedMessages++
		}
		//unlock message transferring
		// nodemem_mutex.Unlock()
		return true
	})
	return actual_bandwidth, droppedMessages
}

// transfers all messages between the nodes
func (al *AddressingLogic) HandleEncounter(config *model.Config, encounter *model.Encounter) (float32, float32) {

	// get the two node IDs involved
	nodeid1 := encounter.Node1
	nodeid2 := encounter.Node2

	// get each node's message map
	messageMap1 := GetMessageQueue(nodeid1)
	messageMap2 := GetMessageQueue(nodeid2)

	band1, drops1 := al.HandleHelper(config, encounter, messageMap1, messageMap2, nodeid1, nodeid2)
	band2, drops2 := al.HandleHelper(config, encounter, messageMap2, messageMap1, nodeid2, nodeid1)

	return band1 + band2, drops1 + drops2
}

// get the logic name
func (al *AddressingLogic) GetLogicName() string {
	return "addressing"
}