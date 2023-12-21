package logic

import (
	"database/sql"
	"errors"
	model "marathon-sim/datamodel"
	"math"
	"strconv"
	"strings"
	"sync"

	logger "github.com/sirupsen/logrus"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

// our logger
var logg *logger.Logger

// Lock the map for exclusive access
var LocksMapMutex *sync.Mutex

// lock sync.map
var lockMessage map[string]*sync.Mutex

var receivedmessageDBChan chan *model.DeliveredMessageDB
var messageDBChan chan *model.MessageDB

// defines the interface for Node Logic
type Logic interface {
	InitLogic(block *LogicConf, log *logger.Logger)
	// every time a node finds itself in a new position, this message is called.
	// `b` is the marshalled form of the location, and will need to be
	// unmarshalled by this method.  See examplelogic.go for an example.
	NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte)

	// potentially does a message exchange between nodes when an encounter
	// occurs. This function returns the ideal bandwidth and actual bandwidth
	HandleEncounter(encounter *model.Encounter) float32

	//data preparation function
	//this function converts the data
	//into a map of nodeid->XYZ geolocations.
	//This is the returned output.
	DataPreparation(rows *sql.Rows, node_list *sql.Rows, logg *logger.Logger, minevents int) (sync.Map, []model.NodeId)

	//Handel helper
	//this funciton is exposed to set the functionality
	//of the attackers.This function returns the ideal bandwidth and actual bandwidth
	HandleHelper(encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) float32
	//get the logic name
	GetLogicName() string
}

// a map of all of the supported node Logic implementations
var LogicStore map[string]Logic

// initialize the logic engines.
//
// Important: new logic engines need to be added here!
func LogicEnginesInit(log *logger.Logger, simLogicFile string) {
	LogicStore = make(map[string]Logic)
	// populate the LogicStore with all of the available logic engines
	LogicStore["broadcast"] = &BroadcastLogic{}
	LogicStore["adversary"] = &Adversary{}
	LogicStore["randomwalk-v1"] = &RandomWalkLogicV1{}
	LogicStore["addressing"] = &AddressingLogic{}

	//init locks on counter
	lockMessage = make(map[string]*sync.Mutex)
	LocksMapMutex = &sync.Mutex{}
	// create the information block of the logic engines
	block := ConfigLogic(simLogicFile, log)
	// initialize each logic engine
	// for name, l := range LogicStore {
	// 	log.Debugf("initializing logic '%v'", name)
	// 	l.InitLogic(block, log)
	// }
	//init the logic engine for the experiment
	logicEng, err := GetLogicByName(block.LogicName)
	if err != nil {
		log.Info("problem with init logic engine, abort!")
		return
	}
	//set organizer, as it is a part of the logic config
	SetOrganizer(block.Organizer)
	//init the logic engine
	logicEng.InitLogic(block, log)
	//set the log generaly
	logg = log
}

// Acquire a lock for a given key
func LockMutexMessage(key string) {

	// Lock the map for exclusive access
	LocksMapMutex.Lock()

	// Check if the lock for the key already exists
	if _, ok := lockMessage[key]; !ok {
		// Create a new lock for the key
		lockMessage[key] = &sync.Mutex{}
	}

	// Acquire the lock for the key
	lockMessage[key].Lock()
	LocksMapMutex.Unlock()
}

func UnlockMutexMessage(key string) {
	// Lock the map for exclusive access
	LocksMapMutex.Lock()

	// Check if the lock for the key exists
	if lock, ok := lockMessage[key]; ok {
		// Release the lock for the key
		lock.Unlock()
	}
	LocksMapMutex.Unlock()
}

func GetInstalledLogicEngines() []string {
	enginesArr := make([]string, 0, len(LogicStore))
	for k := range LogicStore {
		enginesArr = append(enginesArr, k)
	}
	return enginesArr
}

func GetLogicByName(name string) (Logic, error) {
	logic, ok := LogicStore[name]
	if !ok {
		return nil, errors.New("logic engine not found")
	} else {
		return logic, nil
	}
}

// check that an int is not in a slice
func NotInSlice(slice []model.NodeId, num model.NodeId) bool {
	for _, value := range slice {
		if value == num {
			return false
		}
	}
	return true
}

// this function plots the XYZ data into a graph
func PlotData(rows *sql.Rows) {
	// Initialize plot
	p := plot.New()

	// Create plot data points from rows
	var pts plotter.XYZs
	//count := 0
	for rows.Next() {
		var x, y, z float64
		err := rows.Scan(&x, &y, &z)
		if err != nil {
			logg.Fatal(err)
		}
		pts = append(pts, plotter.XYZ{X: x, Y: y, Z: z})
		//count++
		//if count > 1000000 {
		//	break
		//}
	}

	// Add plot data to plot
	scatter, err := plotter.NewScatter(pts)
	if err != nil {
		logg.Fatal(err)
	}
	p.Add(scatter)

	// Set axis labels and tick marks
	p.X.Label.Text = "X Axis"
	p.Y.Label.Text = "Y Axis"
	p.X.Tick.Marker = plot.DefaultTicks{}
	p.Y.Tick.Marker = plot.DefaultTicks{}

	// Save plot to file
	if err := p.Save(600, 400, "plot.png"); err != nil {
		logg.Fatal(err)
	}
}

// assign the channles from the simulate function
func AssignChannels(messageDBChan_ chan *model.MessageDB, messagedeliveredDBChan_ chan *model.DeliveredMessageDB) {
	messageDBChan = messageDBChan_
	receivedmessageDBChan = messagedeliveredDBChan_
}

//this function places the message in the sync.map of recepient
//and updating the db on it

func TransferMessage(encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId, messageId string, message *Message) bool {

	//check that the time did not pass,
	//and the amount of legal hops transfer
	// was not finished
	if message.TTLHops < 0 || encounter.Time-message.CreationTime < float64(message.TTLSecs) {
		return false
	}
	// if it's not present on node 2, make a copy and store it on the node
	if _, ok := messageMap2.Load(messageId); !ok {
		//check if the message was transfered by the nearby node,
		//if so - do not transfer!
		if len(message.path) > 1 && message.path[len(message.path)-2].prevNode == nodeid2 {
			return false
		}
		// LockMutexMessage(messageId)
		// val, _ := MessagesCopies.Load(messageId)
		// MessagesCopies.Store(messageId, val.(int)+1)
		// UnlockMutexMessage(messageId)
		//create a copy
		newMessage := message.Copy()
		// decrease the amount of hops due to transfer
		newMessage.TTLHops--
		newMessage.LatHops++

		newMessage.RecordHop(nodeid2, encounter.Time)

		//messageMap2.Store(message.MessageId, newMessage)
		//update the saving of the message in the buffer
		UpdateMemoryNode(nodeid2, newMessage)

		//add the recieving node to the path
		path_tmp := newMessage.GetPathString()
		//if dest is a nodid, change it to string
		//for document purposes
		dest := newMessage.GetDestinationString()

		//messageDBChan <- m

		//check if the message arrived at destination (approx.)
		//check if message arrived at destination
		if ArrivalCheck(dest, encounter, message, nodeid2) && !message.IsDeliveredYet() { //message arrived at destination
			//document received message at destination
			dm := &model.DeliveredMessageDB{
				ExperimentName: encounter.ExperimentName,
				MessageId:      newMessage.MessageId,
				Sender:         model.NodeIdInt(newMessage.Sender),
				Destination:    dest,
				Payload:        newMessage.Payload.(string),
				DeliverTime:    encounter.Time - newMessage.CreationTime,
				Path:           path_tmp,
				Hops:           newMessage.LatHops,
				FakeMessage:    newMessage.FakeMessage,
				TTLHops:        newMessage.TTLHops,
				TTLSecs:        newMessage.TTLSecs,
				Size:           newMessage.Size,
			}
			receivedmessageDBChan <- dm
			//sign that the message was delivered
			newMessage.MessageDelivered()
		}
		// check for reassmeblement
		if newMessage.ShardsAvailable == 1 {
			//create the data to send to messages DB
			m := &model.MessageDB{
				ExperimentName: encounter.ExperimentName,
				MessageId:      newMessage.MessageId,
				Sender:         model.NodeIdInt(newMessage.Sender),
				Type:           addressTypeToString(newMessage.Type),
				Destination:    dest,
				Payload:        newMessage.Payload.(string),
				CreationTime:   newMessage.CreationTime,
				TransferTime:   encounter.Time,
				Path:           path_tmp,
				Hops:           newMessage.LatHops,
				Sender_Node:    model.NodeIdInt(nodeid1),
				Reciever_Node:  model.NodeIdInt(nodeid2),
				MPop:           newMessage.MPop,
				LPop:           newMessage.LPop,
				FakeMessage:    newMessage.FakeMessage,
				TTLHops:        newMessage.TTLHops,
				TTLSecs:        newMessage.TTLSecs,
				Size:           newMessage.Size,
			}
			dm := &model.DeliveredMessageDB{
				ExperimentName: encounter.ExperimentName,
				MessageId:      newMessage.MessageId,
				Sender:         model.NodeIdInt(newMessage.Sender),
				Destination:    dest,
				Payload:        newMessage.Payload.(string),
				DeliverTime:    encounter.Time - newMessage.CreationTime,
				Path:           path_tmp,
				FakeMessage:    newMessage.FakeMessage,
				TTLHops:        newMessage.TTLHops,
				Hops:           newMessage.LatHops,
				TTLSecs:        newMessage.TTLSecs,
				Size:           newMessage.Size,
			}
			ReassmbleMessage(messageMap2, dm, m, newMessage.CreationTime)
		}

		//message transfered
		return true
	}
	//message already exists,sign that
	//you don't transfer it
	return false
}

// a function that checks if the message arrived at a target
func ArrivalCheck(dest string, encounter *model.Encounter, mes *Message, destnode model.NodeId) bool {

	//if the destination is a geolocation, check if the geolocation arrived
	//and the node id as well
	if strings.Contains(dest, ",") {
		return GeoArrivalCheck(dest, encounter) && NodalArrivalCheck(model.NodeIdString(mes.DestinationNode), destnode)
	} else {
		//if the destination is a nodeid, check if it was found
		return NodalArrivalCheck(dest, destnode)
	}
}

// a function that checks if the message arrived at a target node
func NodalArrivalCheck(dest string, receiver model.NodeId) bool {
	//if the destination is a nodeid, check if it was found
	if model.NodeIdString(receiver) == dest {
		return true
	} else {
		return false
	}
}

// function that checks if the message arrived at a target geolocation
func GeoArrivalCheck(Destination string, encounter *model.Encounter) bool {
	loc, ok := Profilier.LocationProfile.Load(encounter.Node2)
	//error in location getter
	if !ok {
		logg.Infof("error in location access for nodeid %v, aborting threshold calc", model.NodeIdString(encounter.Node2))
		return true
	}
	enc_coord := []float64{float64(encounter.X), float64(encounter.Y), float64(encounter.Z)}
	Address_cur := LocateGridSquare(enc_coord, Profilier.GridData.Gridsize, loc.(map[[3]int]int), Profilier.GridData.Distance)
	// check whether the string is a destination
	parts := strings.Split(Destination, ",")
	var nums [3]float64
	for i, part := range parts {
		intVal, err := strconv.ParseFloat(part, 64)
		if err != nil {
			logg.Info("error converting the destination")
			// handle error: string contains non-float characters
		}
		nums[i] = intVal
	}
	geo := []float64{nums[0], nums[1], nums[2]}
	Address_mes := [3]int{int(geo[0] - float64(Profilier.MinX)/Profilier.GridData.Gridsize), int(geo[1] - float64(Profilier.MinX)/Profilier.GridData.Gridsize), int(geo[2] - float64(Profilier.MinX)/Profilier.GridData.Gridsize)}
	//check if the destination is close enough
	return Distance(Address_cur, Address_mes, Profilier.GridData.Gridsize) <= Profilier.GridData.Distance
}

// data preparation
func OldDataPreparationGeneral(rows *sql.Rows, node_list *sql.Rows, logg *logger.Logger) (sync.Map, []model.NodeId) {
	nodes_timely_data := sync.Map{}
	nodes_map := make(map[model.NodeId]bool)
	// Step 1: arrange the data
	for rows.Next() {
		var ev model.Event
		if err := rows.Scan(&ev.DatasetName, &ev.Time, &ev.Node, &ev.MarshalledLocation, &ev.X, &ev.Y, &ev.Z); err != nil {
			logg.Warnf("invalid event %v", err)
			continue
		}
		//get correct node's time data
		node_data, ok := nodes_timely_data.Load(ev.Node)
		if !ok { //don't have data on this node yet
			node_data = make([]model.Event, 0)
		}
		node_data = append(node_data.([]model.Event), ev)
		nodes_timely_data.Store(ev.Node, node_data)
		nodes_map[ev.Node] = true
	}
	//erase small nodes

	//get the nodes
	nodes := make([]model.NodeId, 0)
	// get list of nodes
	//delete nodes with less than 50 events
	for k := range nodes_map {
		node_data, ok := nodes_timely_data.Load(k)
		if !ok {
			continue
		}
		if len(node_data.([]model.Event)) > 50 {
			nodes = append(nodes, k)
		} else {
			nodes_timely_data.Delete(k)
		}
	}
	randIdx := model.Perm(len(nodes))
	// Create a new slice with the same elements in a random order
	rand_nodes_name_slice := make([]model.NodeId, len(nodes))
	for i, idx := range randIdx {
		rand_nodes_name_slice[i] = nodes[idx]
	}
	//!!erase when you want ranomity!!
	rand_nodes_name_slice = nodes
	// Step 3: split the nodes to profile and general locations
	split_node := int(math.Ceil(0.9 * float64(len(nodes))))
	newnodes := rand_nodes_name_slice[:split_node]
	oldnodes := rand_nodes_name_slice[split_node:]
	//delete the oldnodes data
	for _, node := range oldnodes {
		nodes_timely_data.Delete(node)
	}
	for key := range newnodes {
		node_data, ok := nodes_timely_data.Load(key)
		if !ok {
			continue
		}
		// Get the length of the slice
		data := node_data.([]model.Event)
		originalLen := len(data)

		// Calculate the index up to which we want to keep the elements
		endIdx := int(0.1 * float64(originalLen))
		//sort the slice according to time, the last element
		// Use slicing to create a new slice with only the first 30% of the original elements
		//save the other data for the experiment
		left_items := data[endIdx:]
		//save the first and last time of each node
		nodes_timely_data.Store(key, left_items)
	}
	//return the sorted data
	return nodes_timely_data, newnodes
}

// data preparation
func DataPreparationGeneral(rows *sql.Rows, node_list *sql.Rows, logg *logger.Logger, minevents int) (sync.Map, []model.NodeId) {
	nodes_timely_data := sync.Map{}
	nodes_map := make(map[model.NodeId]bool)
	// Step 1: arrange the data
	for rows.Next() {
		var ev model.Event
		if err := rows.Scan(&ev.DatasetName, &ev.Time, &ev.Node, &ev.MarshalledLocation, &ev.X, &ev.Y, &ev.Z); err != nil {
			logg.Infof("invalid event %v", err)
			continue

		}

		//get correct node's time data
		node_data, ok := nodes_timely_data.Load(ev.Node)
		if !ok { //don't have data on this node yet
			node_data = make([]model.Event, 0)
		}
		node_data = append(node_data.([]model.Event), ev)
		nodes_timely_data.Store(ev.Node, node_data)
		nodes_map[ev.Node] = true
	}
	//erase small nodes

	//get the nodes
	nodes := make([]model.NodeId, 0)
	// get list of nodes
	//delete nodes with less than 50 events
	for k := range nodes_map {
		node_data, ok := nodes_timely_data.Load(k)
		if !ok {
			continue
		}
		if len(node_data.([]model.Event)) > minevents {
			nodes = append(nodes, k)
		} else {
			nodes_timely_data.Delete(k)
		}
	}
	//return the sorted data
	return nodes_timely_data, nodes
}

// Helper function to calculate the distance between two grid positions
func Distance(pos1, pos2 [3]int, gridSize float64) float64 {
	dx := float64(pos1[0]-pos2[0]) * gridSize
	dy := float64(pos1[1]-pos2[1]) * gridSize
	dz := float64(pos1[2]-pos2[2]) * gridSize
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// stores a message at a node; does NOT make a copy of that message
func OriginateMessage(nodeid model.NodeId, message *Message) {
	//update a new message
	UpdateMemoryNode(nodeid, message)
}

// get a message queue by a nodeid
func GetMessageQueue(nodeid model.NodeId) *sync.Map {
	//load the memory of the node
	node_memory, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		logg.Infof("can't get the message queue of node %v", nodeid)
	}
	//get the message queue return it
	messages := node_memory.(*NodeMemory).MessagesQueue
	return messages
}
