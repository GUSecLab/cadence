package main

import (
	"database/sql"
	"errors"
	model "marathon-sim/datamodel"
	logics "marathon-sim/logic"
	"os"
	"sort"
	"strconv"
	"strings"

	"sync"
	"time"

	"gorm.io/gorm"
)

var logic logics.Logic
var dataset model.Dataset
var experimentName string

// number of epochs that will be simulated; global var so that simReporter can access it
var numEpochsToSimulate int

var encounterChan chan *model.Encounter
var bandChan chan *model.Bandwidths
var messageDBChan chan *model.MessageDB
var receivedmessageDBChan chan *model.DeliveredMessageDB

// mutex to serialize DB attempts to find start,end times for nodes
var nodeTimeMutex sync.Mutex

var prefetch *prefetcher

// this is a map where the key is the nodeid and the value is the start and end
// events for the current epoch
var activeNodes *sync.Map

type Epoch struct {
	now  float64
	prev float64
	next float64
}
type startEndEvents struct {
	start *model.Event
	end   *model.Event
}

type NodeIDs []model.NodeId
type NodeIDMap map[model.NodeId]bool

// getTimesFromDB calculates a list of times based on the given start and end times.

// It populates the provided timeChan channel with the retrieved times.
// The time_step parameter specifies the increment between consecutive times.
func getTimesFromDB(startTime, endTime float64, timeChan chan float64, timeStep float32) {

	allTimes := make([]float64, 0)

	// Convert timeStep to float64 for consistency in calculations
	timeStep64 := float64(timeStep)

	// Start the loop at the startTime and increment by timeStep until endTime is reached
	for i := startTime; i <= endTime; i += timeStep64 {
		allTimes = append(allTimes, i)
	}

	// One epoch needs three varibles; prev, now and next. The epoch cannot be created with only prev or now.
	numEpochsToSimulate = len(allTimes) - 2
	log.Infof("finished grabbing times; there are %v epochs to simulate", numEpochsToSimulate)
	for _, t := range allTimes {
		timeChan <- t
	}
	log.Info("no more times to simulate")
	close(timeChan)
}

func isSyncMapEmpty(m *sync.Map) bool {
	empty := true
	m.Range(func(key, value interface{}) bool {
		empty = false
		return false // Stop iterating, as we only need to check if it's not empty
	})
	return empty
}

// gets the first and last time this node was seen in the DB
func getNodeTimeBoundaries(id model.NodeId) (firstAppeared, lastAppeared float64) {
	nodeTimeMutex.Lock()
	defer nodeTimeMutex.Unlock()

	rows, err := model.DB.Table("events").Select("min(time),max(time)").Where("dataset_name=? and node=?", dataset.DatasetName, id).Rows()
	if err != nil || !rows.Next() {
		log.Panicf("unable to find node %v in DB; err=%v", id, err)
	}
	rows.Scan(&firstAppeared, &lastAppeared)
	rows.Close()
	return
}

// gets the nodes from the dataset, considering only the top `ktop_nodes` if `ktop_nodes` > 1
func getRelevantNodes(ktop_nodes int, datasetName string, startTime, endTime float64) (NodeIDs, NodeIDMap) {

	var nodes NodeIDs
	nodes_map := make(map[model.NodeId]bool, 0)

	if ktop_nodes > 1 {
		// only get the top k nodes
		rows, err := model.DB.Table("events").
			Select("node, COUNT(*) as node_count").
			Where("dataset_name=? and time>=? and time<=?", datasetName, startTime, endTime).
			Group("node").
			Order("node_count DESC").
			Limit(ktop_nodes).
			Rows()

		if err != nil {
			log.Infof("error in fetching the ktop nodes %v", err)
		}
		for rows.Next() {
			var node model.NodeId
			var count int
			if err := rows.Scan(&node, &count); err != nil {
				log.Infof("invalid node: %v", err)
				continue
			}
			nodes = append(nodes, node)
			nodes_map[node] = true
		}

	} else { // otherwise, get all nodes
		rows, err := model.DB.Table("events").Select("distinct node").Where("dataset_name=? and time>=? and time<=?", datasetName, startTime, endTime).Rows()
		if err != nil {
			log.Fatal("error in fetching the nodes:", err)
		}

		for rows.Next() {
			var node model.NodeId
			if err := rows.Scan(&node); err != nil {
				log.Infof("invalid node: %v", err)
				continue
			}
			nodes = append(nodes, node)
			nodes_map[node] = true
		}
	}

	if len(nodes) < 2 {
		log.Fatalf("too few nodes in dataset (maybe bad time range?); num_nodes=%v", len(nodes))
	}

	// returns rows so data preparation may use raw db data
	return nodes, nodes_map
}

// This is the function that actually does the hard work of a node.  It figures
// out its position and waits for all other workers to do the same.  It then
// figures out its distance to all other active nodes.
func encounterWorker(
	datasetName string, epochChan chan *Epoch,
	doneBarrier, positionsKnown *sync.WaitGroup, id model.NodeId,
	conditions model.EncounterConditions, alternative_start float64, isPPBR bool) {

	//starting a worker
	log.Infof("starting node worker for node %v", id)

	// first, figure out when this node first and last appears in the dataset
	firstAppeared, lastAppeared := getNodeTimeBoundaries(id)
	log.Debugf("node %v, start %v, end %v", id, firstAppeared, lastAppeared)


	// NOTE : NEW LOGIC HERE FOR DP, JAN 2025 
		//alternative start of node data
	//due to innovative logic engine (DP)
	
	if alternative_start != -1 {
		firstAppeared = alternative_start
	}


	// next, wait for epochs to come in fron the epochChan.  When a new epoch
	// arrives, indicating that there's a new epoch to consider, first compute
	// the start and end positions of this node for the epoch.  Then, wait for
	// all other nodes to do the same.  And finally, compute the distances to
	// each node to figure out if there are any encounters.
	var timeOfLastPositionUpdate float64

	// figure out the distance at which we'll consider two nodes to have
	// an encounter

	distance := DEFAULT_DISTANCE

	for _, condition := range conditions {
		if condition.GetType() == model.ConditionTypeDistanceInMeters {
			distance = condition.(*model.DistanceCondition).GetDistance()
		}
	}

	// repeat for each new epoch; wait for epoch to begin by receiving on the
	// `epochChan`
	for epoch := range epochChan {
		var timeBounds *startEndEvents
		// if you first appeared after the epoch or last appeared before this
		// epoch even began, then this worker shouldn't bother doing anything
		// NOTE: MICAH CHANGED THIS ON 2024.04.30.  THE ORIGINAL VERSION IS
		// COMMENTED OUT ON THE NEXT LINE.  TODO.  VERIFY THAT THIS IS CORRECT.
		// SEE ALSO DISCUSSION AT https://mattermost.cs.georgetown.edu/seclab/pl/hexr83b6s3dk7x8wjcqkytwufe

		// NOTE: this ordering seems to be where the "issue"
		// we would like to only consider nodes that both
		// 1, first appeared before the epoch started
		// 2, last appeared after the epoch ended
		// and this conditional seems to capture that nicely,
		// although it is still unclear why micah's conditional
		// leads to such big differences.

		if firstAppeared >= epoch.prev || lastAppeared <= epoch.now {

			// this line below is micah's code

			// if firstAppeared >= epoch.now || lastAppeared <= epoch.prev {

			//log.Debugf("node %v has no position at epoch %v because its times are %v to %v", id, *epoch, firstAppeared, lastAppeared)
			// activeNodes.Store(id, nil) // not active during this epoch
			positionsKnown.Done()
			doneBarrier.Done()
			continue
		}
		// otherwise, find the position of this node at the start and end of
		// the epoch
		timeBounds, _ = prefetch.getNodePositions(epoch, id)
		if timeBounds == nil {
			//log.Debugf("node %v has no position at epoch %v because it has no timebounds", id, *epoch)
			positionsKnown.Done()
			doneBarrier.Done()
			continue
		}

		// print the time bounds node positions for debugging purposes 
		// indicate that we're active during this epoch
		activeNodes.Store(id, timeBounds)
		
		// inform the node logic that we have a new position
		if timeOfLastPositionUpdate != timeBounds.start.Time {
			// print the dataset coordtype 
			log.Infof("TimeBounds Start Event: %+v", timeBounds.start)
			log.Infof("CoordType for dataset: %v", dataset.CoordType)
			log.Infof("TimeBounds Start MarshalledLocation: %v", timeBounds.start.MarshalledLocation)
			logic.NewPositionCallback(id, dataset.CoordType, timeBounds.start.MarshalledLocation)
			timeOfLastPositionUpdate = timeBounds.start.Time
		}

		// effectively, tell other workers that we're done figuring out our
		// position
		positionsKnown.Done()

		// next, wait for all nodes to get their positions
		positionsKnown.Wait()

		//log.Debug("I think everyone else knows their positions")

		// compute distances to other nodes. Note that the Range function
		// executes the function on every element in the map.
		// TODO: MICAH STOPPED HERE. The anonymous function

		activeNodes.Range(func(k, v any) bool {
			return activeNodesEncounter(k, v, epoch, timeBounds, conditions, datasetName, id, distance, isPPBR)
		})

		doneBarrier.Done() // signal that we're done
	}
}

// this function runs the simulation experiment
func simulate(config *model.Config, investName string) {

	// extract all parameters from the config structure
	logicName := config.Simulation.Logic
	messagesFile := config.Simulation.MessagesFile
	conditionsFile := config.Simulation.ConditionsFile
	startTime := float64(config.Simulation.StartTime)
	endTime := float64(config.Simulation.EndTime)
	datasetName := config.Simulation.DatasetName
	expName := config.Simulation.ExperimentName
	min_message_size := config.Simulation.MinMessageSize
	max_message_size := config.Simulation.MaxMessageSize
	sim_n_splits := config.Simulation.NEncountersSplit
	time_step := config.Simulation.TimeStep
	messageGeneratorOption := strconv.Itoa(config.Simulation.GenerationType)
	messagesTemplatePath := config.Simulation.MessagesTemplate
	//(old message generation logic) generatorScriptPath := config.Simulation.GeneratorScript
	//dbFile := config.TopLevel.DBFile
	
	// TODO: ktop nodes is no longer used, is this correct?
	//ktop_nodes := config.Simulation.KTopNode

	// new for DP logic? 
	multiplier_timestep := 10.0

	// store params in an ExperimentConfig structure

	experimentConfig := model.CopyConfig(config)

	if r := model.DB.Create(&experimentConfig); r.Error != nil {
		log.Warn("cannot create experiment configuration record")
		log.Warn("Recommendation: please check for valid parameters and try again")
		log.Fatalf("cannot create experiment configuration record: %v", r.Error)
	}

	//get logic engine
	var err error

	logic, err = logics.GetLogicByName(logicName)
	if err != nil {
		log.Fatalf("cannot retrieve logic engine with name %v: %v", logicName, err)
	}

	// initialize the delivered messages queue
	logics.DeliveredMessagesQueue = model.NewFastQueue()

	// set conditions
	conditions, conditionsDesciption, err := model.ReadConditionsFile(conditionsFile)
	if err != nil {
		log.Fatalf("cannot read conditions file (%v): %v", conditionsFile, err)
	}

	// retrieve the dataset from the database
	r := model.DB.First(&dataset, "dataset_name=?", datasetName)
	if r.Error != nil || r.RowsAffected != 1 {
		log.Fatalf("error retrieving dataset with name %v", datasetName)
	}

	// create an experiment and save it in the DB
	experimentName = expName
	exp := &model.Experiment{
		ExperimentName:     expName,
		DatasetName:        datasetName,
		Investigator:       investName,
		DistanceConditions: conditionsDesciption,
		DateStarted: sql.NullTime{
			Time:  time.Now().UTC(),
			Valid: true,
		},
	}

	if r := model.DB.Create(&exp); r.Error != nil {
		log.Warn("cannot create experiment.  this is likely because an experiment with the same name already exists")
		log.Warn("Recommendation: change the experiment name to something new, and try again")
		log.Fatalf("cannot create experiment: %v", r.Error)
	}

	// starting simulation

	log.Infof("beginning simulation on dataset %v; will run from %v to %v",
		datasetName, startTime, endTime)
	log.Infof("recording results to %v", expName)

	/*

	// Jan 2025 - this is the old logic, logic for getting relevant nodes 
	// was encapsulated in the getRelevantNodes function
	nodes, nodes_map := getRelevantNodes(ktop_nodes, datasetName, startTime, endTime)

	*/

	// preparing the data for the addressing logic engines
	// TODO - modify getRelevantNodes to accomodate the new addressing logic

	var nodes []model.NodeId
	var times_data *sync.Map
	var is_timely_data bool

	//prepare the data for the addressing logic engines
	if strings.HasPrefix(logic.GetLogicName(), "addressing") || strings.HasPrefix(logic.GetLogicName(), "mirage") {
		rows2, err2 := model.DB.Table("events").Select("time,node,x,y,z").Where("dataset_name=? and time>=? and time<=?", datasetName, startTime, endTime).Order("time").Rows()
		if err2 != nil {
			log.Errorf("error in fetching the events for profiling %v", err2)
		}

		// raise a flag that the assignment of node->district occurred now
		is_timely_data = true
		// check for the regular HumaNets engine
		if _, ok := logic.(*logics.AddressingLogic); ok {

			times_data, nodes = logic.(*logics.AddressingLogic).DataPreparation(rows2, log, 0, config)

		} else {
			// check for the HumaNets-DP engine
			if _, ok := logic.(*logics.MirageLogic); ok {
				times_data, nodes = logic.(*logics.MirageLogic).DataPreparation(rows2, log, 0, config)
			}
		}
		rows2.Close()

	} else {
		
		// TODO: encapsulate this into a new getRelevantNodes function

		rows, err := model.DB.Table("events").Select("distinct node").Where("dataset_name=? and time>=? and time<=?", datasetName, startTime, endTime).Rows()
		if err != nil {
			log.Infof("error in fetching the nodes")
		}

		for rows.Next() {
			var node model.NodeId
			if err := rows.Scan(&node); err != nil {
				log.Infof("invalid node: %v", err)
				continue
			}
			nodes = append(nodes, node)
		}
		if len(nodes) < 2 {
			log.Fatalf("too few nodes in dataset (maybe bad time range?); num_nodes=%v", len(nodes))
		}
		rows.Close()
	}


	//init the message status monitor
	model.MonitorInit()

	// allocate some stuff
	activeNodes = new(sync.Map)
	encounterChan = make(chan *model.Encounter, 1000)                  // buffer size of 1000 is arbitrary
	messageDBChan = make(chan *model.MessageDB, 1000)                  // buffer size of 1000 is arbitrary
	receivedmessageDBChan = make(chan *model.DeliveredMessageDB, 1000) // buffer size of 1000 is arbitrary
	EpochLoadChan := make(chan *model.EpochLoad, 1000)                 // buffer size of 1000 is arbitrary
	bandChan = make(chan *model.Bandwidths, 1000)                      // channel for bandwidth documnetation
	prefetch = new(prefetcher)                                         // prefetcher fetches events associated with a node in the database
	prefetch.init()
	//send the relevant channels to logics
	logics.AssignChannels(messageDBChan, receivedmessageDBChan)
	// kick off the encounter recorder as a separate goroutine.  this goroutine
	// will just wait for data (encounters) sent via the encounterChan, and then
	// will record them in the DB (in the Encounters table)
	var recorderSyncBarrierDB sync.WaitGroup
	var recorderSyncBarrierRecievedDB sync.WaitGroup
	var recorderSyncBarrierEpochLoadsDB sync.WaitGroup
	var recorderSyncBarrierBandwidth sync.WaitGroup

	//sign that encounters creation is complete
	//as it done through epoch
	//create the data to send to messages DB
	//this is the first part - before the encounters
	distance := DEFAULT_DISTANCE //default distance
	for _, condition := range conditions {
		if condition.GetType() == model.ConditionTypeDistanceInMeters {
			distance = condition.(*model.DistanceCondition).GetDistance()
		}

	}

	// determine if we already computed the encounters for this dataset and distance
	encountersExists := true
	isPPBR := false 

	if strings.HasPrefix(logic.GetLogicName(), "addressing") {
		isPPBR = true
	}

	var rowInfo model.DatasetEncounterEnumeration // create a variable to hold the result; not really used
	// Check if a row with the specified values exists

	// if we are using PPBR, we must have valid PPBR encounters 
	// if we are not using PPBR, we can have whatever encounters 

	if isPPBR {
		if err := model.DB.Where("distance = ? AND dataset_name = ? AND complete = ? AND duration=? AND PPBR=?", distance, datasetName, true, time_step, isPPBR).First(&rowInfo).Error; err != nil {
			// Handle the case where the row does not exist
			if errors.Is(err, gorm.ErrRecordNotFound) {
				encountersExists = false
			} else {
				log.Fatalf("Error checking for an existing row: %v", err)
			}
		}

	} else {
		if err := model.DB.Where("distance = ? AND dataset_name = ? AND complete = ? AND duration=?", distance, datasetName, true, time_step).First(&rowInfo).Error; err != nil {
			// Handle the case where the row does not exist
			if errors.Is(err, gorm.ErrRecordNotFound) {
				encountersExists = false
			} else {
				log.Fatalf("Error checking for an existing row: %v", err)
			}
		}
	}

	// if we didn't already compute the encounters, then let's compute them!
	if !encountersExists {
		log.Info("finding encounters")
		//findEncounters(nodes, datasetName, conditions, startTime, endTime, time_step, distance)
		findEncountersDP(nodes, datasetName, conditions, startTime, endTime, time_step, distance, is_timely_data, times_data, isPPBR)

	}

	//message generation
	//first, check if we need to create it
	//if it exists, use it
	//elsewhere, create the file
	if info, err := os.Stat(messagesFile); err == nil && !info.IsDir() {
		log.Info("The message file exists. skip creation")
	} else {
		log.Info("The message file doesn't exists. create it")
		//Prepare for the message generation, parse the template file and save sender and receiver ndoes.
		ktop_nodes := -1
		messageGenerate(datasetName, messagesTemplatePath, min_message_size, max_message_size,
			startTime, endTime, messagesFile, messageGeneratorOption, distance, ktop_nodes)
	}

	//checking if there are nodes to remove from messaging
	//if os, do it
	var messages map[model.NodeId][]*logics.Message
	//load the messages
	messages, err = logics.LoadMessages(messagesFile, log, logic)
	if err != nil {
		log.Fatal("cannot load messages", err)
	}
	//throughput calculation
	//first-gather the amount of messages
	
	// modified to accomodate DP/humanets
	mes_number := 0
	if strings.HasPrefix(logic.GetLogicName(), "addressing") { //related only for addressing
		for n := range messages {

			if contains(nodes, n) { //add only the node ids that are relevant
				mess := removeNilsMes(messages[n])
				mes_number += len(mess)
				messages[n] = mess
			}
		}

	} else { //no dp/humanets protocol, add all messages
		for n := range messages {

			mes_number += len(messages[n])
		}
	}


	start_after_encounters := time.Now()
	//channel for bandwidths recording
	go model.RecordBandwidth(experimentName, bandChan, &recorderSyncBarrierBandwidth)
	recorderSyncBarrierBandwidth.Add(1)
	//add a channel for message recording, like the encounter table
	go logics.RecordMessages(experimentName, messageDBChan, &recorderSyncBarrierDB)
	recorderSyncBarrierDB.Add(1)
	//add a channel for message recording, like the encounter table
	go logics.RecordDeliveredMessages(experimentName, receivedmessageDBChan, &recorderSyncBarrierRecievedDB)
	recorderSyncBarrierRecievedDB.Add(1)
	//record epoch loads
	go model.RecordEpochLoad(experimentName, EpochLoadChan, &recorderSyncBarrierEpochLoadsDB)
	recorderSyncBarrierEpochLoadsDB.Add(1)
	// init memory of nodes
	for _, n := range nodes {
		// log.Infof("initializing node %v memory", n)
		logics.InitNodeMemory(config, n)
	}
	//for this case, we iterate the list of encounters and simulate the message transfer
	//get the encounters

	var encounters []model.Encounter

	const batchSize = 100000
	offset := 0
	for {

		log.Infof("fetching encounters from DB, offset %v", offset)

		if isPPBR {
			log.Infof("fetching encounters for PPBR logic")
			rows, err := model.DB.Table("encounters").
				Select("*").
				Where("dataset_name=? and distance=? and PPBR=?", datasetName, distance, isPPBR).
				Order("time ASC").
				Offset(offset).
				Limit(batchSize).
				Rows()
	
			if err != nil {
				log.Infof("error in fetching the events for profiling")
				break
			}
	
			if !rows.Next() {
				rows.Close()
				break
			}


		
			simulateBatchEncounters(config, encounters, rows, sim_n_splits, multiplier_timestep, time_step, startTime, endTime, expName, EpochLoadChan, messages, logic)

			// update the offset 
			offset += batchSize

			// Close the rows after processing
			rows.Close()
		} else {
			rows, err := model.DB.Table("encounters").
				Select("*").
				Where("dataset_name=? and distance=?", datasetName, distance).
				Order("time ASC").
				Offset(offset).
				Limit(batchSize).
				Rows()
	
			if err != nil {
				log.Infof("error in fetching the events for profiling")
				break
			}
	
			if !rows.Next() {
				rows.Close()
				break
			}

		
			simulateBatchEncounters(config, encounters, rows, sim_n_splits, multiplier_timestep, time_step, startTime, endTime, expName, EpochLoadChan, messages, logic)

			// update the offset 
			offset += batchSize

			// Close the rows after processing
			rows.Close()
		}

	}

	/*
	rows, err := model.DB.Table("encounters").Select("*").Where("dataset_name=? and distance=?", datasetName, distance).Order("time ASC").Rows()
	if err != nil {
		log.Infof("error in fetching the events for profiling")
	}

	log.Infof("found %v encounters", len(encounters))

	wasSplit := false //flag to know if sort is needed
	for rows.Next() {
		var enc model.Encounter
		if err := rows.Scan(&enc.DatasetName, &enc.Distance,
			&enc.ExperimentName, &enc.Time,
			&enc.Node1, &enc.Node2, &enc.Duration,
			&enc.X, &enc.Y, &enc.Z); err != nil {
			log.Warnf("invalid encounter %v ", err)
			continue
		}

		// TODO: check this logic - why are we multiplying the time_step by a multipler here?
		// encs := splitEncounter(&enc, float64(sim_n_splits), float64(time_step))
		encs := splitEncounter(&enc, float64(sim_n_splits), float64(float32(multiplier_timestep)*time_step))

		if len(encs) > 1 {
			wasSplit = true
			for _, e := range encs {
				encounters = append(encounters, *e)
			}
		} else {
			encounters = append(encounters, *(encs[0]))
		}

	}
	log.Infof("finished splitting big encounters")
	//sort if needed
	if wasSplit {
		// Define a custom less function that compares Time, node1, and node2 fields.
		lessFunc := func(i, j int) bool {
			if encounters[i].Time != encounters[j].Time {
				return encounters[i].Time < encounters[j].Time
			}
			if encounters[i].Node1 != encounters[j].Node1 {
				return encounters[i].Node1 < encounters[j].Node1
			}
			return encounters[i].Node2 < encounters[j].Node2
		}

		// Sort the encounters slice using sort.Slice and the custom less function.
		sort.Slice(encounters, lessFunc)

	}
	log.Infof("finished sorting new encounters list")

	// timestamp := 0.0
	// iterate all encounters

	//now, if the logic involves geolocation
	//like HumaNets/HumaNets-DP,
	//we need to manipulate the data a little bit
	//get only the encounters relevant the simulation

	encounters = getSubsliceEncounters(encounters, startTime, endTime)
	prev := encounters[0].Time

	// TODO: logic here is new, should compare with old version and logic check 
	for ind, encounter := range encounters {
		if encounter.Time > endTime { //finished this simulation,break
			log.Info("Stop! we fininshed running the experiment.")
			break
		}
		if encounter.Time < startTime { //skip this encounter, too soon
			continue
		}
		//in here, we check if the nodes in the encounter
		//are alive in the timestamp
		if strings.HasPrefix(logic.GetLogicName(), "addressing") || strings.HasPrefix(logic.GetLogicName(), "mirage") {
			//check if one of the nodes does not exists
			//if this is the case, move on 
			var TimesNode1 any
			var TimesNode2 any
			var ok, ok2 bool
			var t1, t2 float64
			if TimesNode1, ok = times_data.Load(encounter.Node1); !ok {
				//log.Infof("node1 %v does not appear in the timely data", encounter.Node1)

				continue
			}
			if TimesNode2, ok2 = times_data.Load(encounter.Node2); !ok2 {
				//log.Infof("node2 %v does not appear in the timely data", encounter.Node2)
				continue
			}
			t1 = TimesNode1.(float64)
			t2 = TimesNode2.(float64)

			//skip encounter if the nodes are not alive again
			if t1 > encounter.Time || t2 > encounter.Time {
				//log.Infof("%v or %v are still asleep at %v", encounter.Node1, encounter.Node2, encounter.Time)
				continue
			}

		}
		//time to calculate the epoch load
		if prev != encounter.Time {
			// calculate the load
			load, avg := logics.GetLoadandAvg()
			// calculate the epochs load and avg
			epl := model.EpochLoad{
				ExperimentName: expName,
				Now:            encounter.Time,
				Prev:           prev,
				Load:           load,
				AvgLoad:        avg,
			}
			prev = encounter.Time
			// //send to the epoch load channel
			EpochLoadChan <- &epl
		}
		//update encounter with experiment
		encounter.ExperimentName = expName
		//call a function that based on an encounter
		//and the messges slice, will simulate
		//the encounter
		log.Infof("Simulating encounter %v. %v encounters to go", ind, len(encounters)-(ind+1))
		encounterSim(&encounter, messages, logic)
	}

	*/
	//empty the encounters manager
	//so that encounters that still wait will finish
	//EmptyEncountnersManager()

	// hurrah!  we're done.  Let's remember to close the encounterChan so that
	// anything left in the buffer is flushed to the DB, and the reporterChan so
	// that simSpeedReporter eventually stops.

	close(bandChan)
	close(messageDBChan)
	close(receivedmessageDBChan)
	close(EpochLoadChan)
	// let's wait for the channels to close
	recorderSyncBarrierDB.Wait()
	recorderSyncBarrierRecievedDB.Wait()
	recorderSyncBarrierEpochLoadsDB.Wait()
	recorderSyncBarrierBandwidth.Wait()

	// finally, let's update the experiment table and note when it is that this
	// experiment finished
	exp.DateFinished.Time = time.Now().UTC()
	exp.DateFinished.Valid = true
	if r := model.DB.Save(&exp); r.Error != nil {
		log.Fatalf("cannot update experiment: %v", r.Error)
	}
	//get max buffer usage
	max_usage, max_node := logics.MaxBufferUsage()
	//document max buffer usage
	//create the data to send to messages DB
	buf_use := &model.BufferMax{
		ExperimentName: expName,
		DatasetName:    datasetName,
		Node:           max_node,
		Max:            max_usage,
	}
	if r := model.DB.Save(&buf_use); r.Error != nil {
		log.Infof("cannot add max buffer usage")
	}
	//save all metrics to one table
	var lat_sec, lat_hops, max_band, band float32
	var del_mes int
	//get the latencies
	latencies_from_table := model.DB.Table("delivered_message_dbs").Select("avg(deliver_time - creation_time),avg(hops)").Where("experiment_name=?", expName).Row()
	err = latencies_from_table.Scan(&lat_sec, &lat_hops)
	if err != nil {
		log.Info(err)
	}

	//get max bandwidth
	max_bandwidth_from_table := model.DB.Table("bandwidths").Select("max(bandwidth)").Where("dataset=? and distance=? and logic=? and experiment_name=?", datasetName, distance, logicName, experimentName).Row()
	err = max_bandwidth_from_table.Scan(&max_band)
	if err != nil {
		log.Info(err)
	}

	//get network load
	netload_from_table := model.DB.Table("bandwidths").Select("sum(bandwidth)").Where("dataset=? and distance=? and logic=? and experiment_name=?", datasetName, distance, logicName, expName).Row()
	err = netload_from_table.Scan(&band)
	if err != nil {
		log.Info(err)
	}

	throughput_from_table := model.DB.Table("delivered_message_dbs").Select("count(*)").Where("experiment_name=?", expName).Row()
	err = throughput_from_table.Scan(&del_mes)
	throuput := float32(del_mes) / float32(mes_number)
	if err != nil {
		log.Info(err)
	}
	//get the load during the experiment
	maxLoad := 0.0
	avgLoad := 0.0
	var nowT float64
	var load float64
	var sum float64
	var count int
	load_during_simulation, err := model.DB.Table("epoch_loads").Select("now,`load`").Where("experiment_name=?", expName).Rows()
	if err != nil {
		log.Info(err)
	}
	// Initialize maxLoad with the first load value
	// Iterate through the remaining rows (if any) and update maxLoad and sum
	for load_during_simulation.Next() {
		if err := load_during_simulation.Scan(&nowT, &load); err != nil {
			// handle error
			log.Info(err)
		}

		// Update maxLoad if necessary
		if load > maxLoad {
			maxLoad = load
		}

		// Update sum and count for calculating average later
		sum += load
		count++
	}
	load_during_simulation.Close()
	// Calculate the average load
	avgLoad = sum / float64(count)

	//save the result
	res_tab := &model.ResultsDB{
		ExperimentName:    expName,
		LatSec:            lat_sec,               //latency in nanoseconds (average)
		LatHop:            lat_hops,              //latency in hops (average)
		MaxBuf:            max_usage,             //max buffer
		MaxBand:           int(max_band),         //max bandwidth
		NetLoad:           int(band),             //network load - sum of all bandwidths
		Throughput:        throuput,              //ratio of delivered messages
		NumMessages:       mes_number,            //total number of messages
		AvgCopiesMessages: logics.GetAvgCopies(), //average amount of message copies at the end of the experiment
		PeakLoad:          maxLoad,               //maximum load (copies of all messages) during experiment
		AverageLoad:       avgLoad,               //average load (copies of all messages) during experiment
	}

	if r := model.DB.Save(&res_tab); r.Error != nil {
		log.Infof("cannot save the results")
	}
	//save the experiment to its family as well
	//save the result
	exp_fam := &model.ExperimentFamily{
		FamilyDataset:  datasetName,
		FamilyDistance: distance,
		ExperimentName: expName,
		Logic:          logicName,
	}

	if r := model.DB.Save(&exp_fam); r.Error != nil {
		log.Infof("cannot save the experiment to its family: %v", r.Error)
	}
	elapsed := time.Since(start_after_encounters)
	log.Infof("Elapsed time of simulation: %v", elapsed.String())
}

/*
// this is the old find encounters function, new logic is 
// findEncountersDP

// this function finds the encounters between nodes
// it launches an encounterWorker goroutine for
// each relevant node
func findEncounters(nodes NodeIDs, datasetName string, conditions model.EncounterConditions, startTime float64, endTime float64, timeStep float32, distance float64) {
	log.Info("creating encounters")

	var recorderSyncBarrier sync.WaitGroup
	var doneBarrier sync.WaitGroup
	var positionsKnown sync.WaitGroup

	// set up WaitGroups
	doneBarrier.Add(len(nodes))
	positionsKnown.Add(len(nodes))

	// kick off a goroutine to save the encounters to the DB, using
	// `recorderSyncBarrer` as a way of communicating that we're done
	// (see end of function where we wait on `recorderSyncBarrier`)
	recorderSyncBarrier.Add(1)
	go model.RecordEncounters(experimentName, encounterChan, &recorderSyncBarrier)

	start_enc_calc := time.Now()
	epochChans := make([]chan *Epoch, len(nodes))

	// for each worker, kick off a goroutine to find encounters.
	// we'll create an "epoch channel" for each of these goroutines
	// so that we can communicate to each one the current epoch
	for i, n := range nodes {

		epochChans[i] = make(chan *Epoch)
		// go encounterWorker(
		go encounterWorker(
			datasetName,
			epochChans[i],
			&doneBarrier,
			&positionsKnown,
			n,
			conditions)
	}

	// create a goroutine that sends to the `timeChan` channel all the times
	// that the simulator will simulate
	timeChan := make(chan float64, 128)
	go getTimesFromDB(startTime, endTime, timeChan, timeStep)

	reporterChan := make(chan float64)
	go simSpeedReporter(reporterChan)

	populateEpochChans(nodes, timeChan, &doneBarrier, &positionsKnown, epochChans, reporterChan)

	close(reporterChan)
	EmptyEncountersManager()

	// save the fact that we computed the encountesr to the
	// DatasetEncounterEnumeration table in the database
	createEncountersEntry := &model.DatasetEncounterEnumeration{
		Distance:    float32(distance),
		Duration:    timeStep,
		DatasetName: datasetName,
		Complete:    true,
	}
	if r := model.DB.Create(createEncountersEntry); r.Error != nil {
		log.Warnf("cannot record that we finished the encounter listing: %v", r.Error)
	}
	elapsed_enc := time.Since(start_enc_calc)
	log.Infof("elapsed time from encounters creation start: %v", elapsed_enc.String())

	close(encounterChan)
	recorderSyncBarrier.Wait()
}

*/

func findEncountersDP(nodes NodeIDs, datasetName string, conditions model.EncounterConditions, startTime float64, endTime float64, timeStep float32, distance float64, is_timely_data bool, times_data *sync.Map, isPPBR bool) {

	log.Info("creating encounters")
	var recorderSyncBarrier sync.WaitGroup
	var doneBarrier sync.WaitGroup
	var positionsKnown sync.WaitGroup
	var epochChans []chan *Epoch

	doneBarrier.Add(len(nodes))
	positionsKnown.Add(len(nodes))
	
	go model.RecordEncounters(experimentName, encounterChan, &recorderSyncBarrier)
	recorderSyncBarrier.Add(1)
	start_enc_calc := time.Now()
	var alternative_start any
	var ok bool
	//get the first timestamp
	//if our
	if is_timely_data {
		// var nodes_enc_list []model.NodeList
		// var nodes_enc []model.NodeId
		// r = model.DB.Find(&nodes_enc_list, "dataset_name=?", datasetName)
		// if r.Error != nil {
		// 	log.Panicf("cannot retrieve list of node names: %v", r.Error)
		// }
		// for _, i := range nodes_enc_list {
		// 	nodes_enc = append(nodes_enc, i.Node)
		// }
		// if len(nodes_enc) < 2 {
		// 	log.Fatalf("too few nodes in dataset (maybe bad time range?); num_nodes=%v", len(nodes))
		// }
		epochChans = make([]chan *Epoch, len(nodes))

		// kick off each node worker
		for i, n := range nodes {
			var alternative_start_float float64
			alternative_start, ok = times_data.Load(n)
			if !ok {
				log.Infof("loading start time of node %v failed", n)
				alternative_start_float = -1.0
			} else {
				alternative_start_float = alternative_start.([]model.Event)[0].Time
			}
			//create the channel for epochs of the node
			epochChans[i] = make(chan *Epoch)
			go encounterWorker(
				datasetName,
				epochChans[i],
				&doneBarrier,
				&positionsKnown,
				n,
				conditions, alternative_start_float, isPPBR)

		}
		//nodes = nodes_enc //update the nodes list
	} else { //no DP involvement
		// kick off each node worker
		epochChans = make([]chan *Epoch, len(nodes))

		for i, n := range nodes {
			//create the channel for epochs of the node
			epochChans[i] = make(chan *Epoch)
			go encounterWorker(
				datasetName,
				epochChans[i],
				&doneBarrier,
				&positionsKnown,
				n,
				conditions, 
				-1.0,
				isPPBR)
		}
	}
	//close the epoch channels

	// fetch times from database
	timeChan := make(chan float64, 1024)
	go getTimesFromDB(startTime, endTime, timeChan, timeStep)

	// kick off the simulation time reporter
	reporterChan := make(chan float64)
	go simSpeedReporter(reporterChan)

	// process epochs as they arrive; they'll arrive from the `getTimesFromDB`
	// function
	var prev, now, next float64

	for t := range timeChan {
		switch {
		case prev == 0:
			prev = t
			continue
		case now == 0:
			now = t
			continue
		case next == 0:
			next = t
			continue
		default:
			epoch := Epoch{
				now:  now,
				prev: prev,
				next: next,
			}

			// send the epoch info (i.e., the start and end times of this epoch)
			// to each node worker
			for i := range nodes {
				epochChans[i] <- &epoch
			}

			// wait for all nodes to finish
			doneBarrier.Wait()
			// reset the barriers for next iteration
			doneBarrier.Add(len(nodes))
			positionsKnown.Add(len(nodes))
			activeNodes = new(sync.Map)
			//prefetch.init() //init bins for next use
			prev = now
			now = next
			next = t
		}

		log.Debugf("simulating time %f", t)
		reporterChan <- t
	}
	close(reporterChan)
	EmptyEncountersManager()

	//update that we finished creating the encounters
	create_encounters := &model.DatasetEncounterEnumeration{
		Distance:    float32(distance),
		Duration:    timeStep,
		DatasetName: datasetName,
		Complete:    true,
		PPBR:        isPPBR,
	}
	if r := model.DB.Create(create_encounters); r.Error != nil {
		log.Infof("cannot sign that we finished the encounter listing")
	}
	elapsed_enc := time.Since(start_enc_calc)
	log.Infof("Elapsed time from encounters creation start: %v", elapsed_enc.String())

	close(encounterChan)
	recorderSyncBarrier.Wait()
}

// Helper function for encounterWorker.
// This function will help to find if two active nodes been encountered between the prev epoch and current epoch.
func activeNodesEncounter(
	k, v any, epoch *Epoch, timeBounds *startEndEvents,
	conditions model.EncounterConditions, datasetName string,
	id model.NodeId, distance float64, isPPBR bool) bool {

	otherNodeId := k.(model.NodeId)
	// let's do these comparisons once, and let's not compare against
	// ourselves.  Arbitrarily, we'll say that nodes i and j will
	// compute distances if i < j.
	if id >= otherNodeId {
		return true // keep going (true for `Range` function means keep going)
	}

	otherNodeTimeBounds := v.(*startEndEvents)

	//log.Infof("computing closest distance between nodes %v and %v in this epoch", id, otherNodeId)
	crossedPaths, t, avX, avY, avZ := didCrossPaths(epoch, timeBounds, otherNodeTimeBounds, conditions,
		nil, nil)

	// if we crossed paths, let's inform the RecordEncounters goroutine
	// to save it to the DB.  We do this by sending the encounter to the
	// encounterChan.
	if crossedPaths {
		//create an encounter
		enc := &model.Encounter{
			DatasetName:    datasetName,
			ExperimentName: experimentName,
			Time:           t,
			Node1:          id,
			Node2:          otherNodeId,
			X:              avX,
			Y:              avY,
			Z:              avZ,
			PPBR:           isPPBR,
		}
		// some conditions may need to update themselves based on the encounter, so
		// call their UpdateEncounterConsequences method.  This is somewhat rare, as
		// most conditions (e.g., distance conditions) don't need to update themselves
		// (in which case this call is a nop)

		for _, condition := range conditions {
			condition.UpdateEncounterConsequences(enc)
		}

		enc.Distance = distance

		// TODO: check this logic
		//log the updated encounter
		CheckEncounter(enc, epoch)
		//encounterChan <- enc
	}
	return true
}

// Helper function for findEncounter.
// This function helps to create epoch and populate it by the time from timeChan.
func populateEpochChans(nodes NodeIDs, timeChan chan float64, doneBarrier *sync.WaitGroup, positionsKnown *sync.WaitGroup, epochChans []chan *Epoch, reporterChan chan float64) {
	var prev, now, next float64

	//if the value in the time channel is less than 3, the epoch can not be created.
	if len(timeChan) < 3 {
		log.Warn("There is not enough time to create an epoch now.")
	}

	//iterate over the values from time channel to create epochs
	for t := range timeChan {
		var epoch Epoch

		//find prev, now and next value and prepare the parameters for an epoch
		switch {
		// Initialization for the epoch. it populates prev, now, and next
		// with the first three values from the timeChan
		case prev == 0:
			prev = t
			continue
		case now == 0:
			now = t
			continue
		case next == 0:
			next = t
		// once we've passed that "initialization" stage, we're going to define
		// an actual epoch and send it to all of the nodes
		default:
			prev = now
			now = next
			next = t
		}

		//create an epoch
		epoch = Epoch{
			prev: prev,
			now:  now,
			next: next,
		}
		//send the value of created epoch to epochs Channel
		for i := range nodes {
			epochChans[i] <- &epoch
		}
		doneBarrier.Wait()

		// reset the barriers for the next epoch (i.e., to wait for all of the
		// workers to do their things)
		doneBarrier.Add(len(nodes))
		positionsKnown.Add(len(nodes))

		// reset the list of active nodes for the next epoch
		activeNodes = new(sync.Map)

		log.Debugf("simulating time %f", t)
		reporterChan <- t
	}

}

// Report the progress and speed of simulated epochs
func simSpeedReporter(timeChan chan float64) {
	const timeDelta = time.Second * 5

	lastTimeWallClock := time.Now()
	lastTimeSimTime := -1.0
	var td time.Duration
	simulatedEpochs := 0
	for t := range timeChan {
		simulatedEpochs += 1
		if lastTimeSimTime > 0 {
			if td = time.Since(lastTimeWallClock); td > timeDelta {
				perDone := 100.0 * float64(simulatedEpochs) / float64(numEpochsToSimulate)
				log.Infof("encounters creation time is %f [simulated %v seconds in %v (wall clock); speedup=%f; %v%% complete]",
					t, t-lastTimeSimTime, td, (t-lastTimeSimTime)/td.Seconds(), perDone)
				lastTimeWallClock = time.Now()
				lastTimeSimTime = t
			}
		} else {
			lastTimeSimTime = t
		}
	}
}

// place a message for nodeid
// and delete the message from the total
// message queue
func setMessagesEncounterPerNode(config *model.Config, messages map[model.NodeId][]*logics.Message, nodeid model.NodeId, encounter model.Encounter) {
	st_time := encounter.Time
	end_time := st_time + float64(encounter.Duration)
	//add the messages
	var tmp_slice_storage []*logics.Message
	for _, mes := range messages[nodeid] {
		//add messages from before the epoch,
		//or before its end

		if mes.CreationTime <= end_time {
			logics.OriginateMessage(config, nodeid, mes)

		} else {
			//add the message to a tmp slice
			//so it can be stored for the rest
			//of the simulation
			tmp_slice_storage = append(tmp_slice_storage, mes)
		}

	}

	// result: if the message was created before the encounter ends, 
	// it is added to the node's message queue, and it is NOT added to the
	// temp slice. the temp slice contains only messages that are NOT YET INVOLVED 
	// in the experiment. 
	
	//save only the messages that weren't added yet
	messages[nodeid] = tmp_slice_storage
}

// simulate the encounter
func encounterSim(config *model.Config, encounter *model.Encounter, messages map[model.NodeId][]*logics.Message, logic logics.Logic) {
	node1 := encounter.Node1
	node2 := encounter.Node2
	//set the messages in the appropriate queues
	setMessagesEncounterPerNode(config, messages, node1, *encounter)
	setMessagesEncounterPerNode(config, messages, node2, *encounter)
	band, drops := logic.HandleEncounter(config, encounter)
	// create the struct to store bandwidth
	b := &model.Bandwidths{
		Dataset:        encounter.DatasetName,
		Distance:       encounter.Distance,
		Logic:          logic.GetLogicName(),
		Bandwidth:      band,
		Drops:          drops,
		Time:           encounter.Time,
		Node1:          node1,
		Node2:          node2,
		Hash:           strconv.FormatUint(model.CalculateHash(encounter.Time, model.NodeIdInt(node1), model.NodeIdInt(node2)), 10),
		ExperimentName: experimentName,
	}
	bandChan <- b
	//update message status of both nodes
	model.MesStatusMonitor.UpdateNodeMessageStatus(node1, logics.GetMessageQueue(node1), node2, logics.GetMessageQueue(node2))
}

// split one encounter to multiple by def'd split of encounters
func splitEncounter(encounter *model.Encounter, splitter float64, encounter_len float64) []*model.Encounter {

	//if the duration is small, return an array with
	//the initial duration
	if float64(encounter.Duration) <= splitter*encounter_len {
		return []*model.Encounter{encounter}
	}

	//else, split the encounter
	// Calculate the number of splits needed.
	numSplits := int(encounter.Duration / float32(splitter*encounter_len))
	// Initialize a slice to hold the split encounters.
	splitEncounters := make([]*model.Encounter, numSplits)

	for i := 0; i < numSplits; i++ {
		// Calculate the start time and duration for each split.
		splitStartTime := encounter.Time + float64(i)*splitter*encounter_len
		splitDuration := splitter * encounter_len

		// Create a new split encounter and add it to the slice.
		splitEncounters[i] = encounter.Copy()

		splitEncounters[i].Time = splitStartTime
		splitEncounters[i].Duration = float32(splitDuration)
	}
	return splitEncounters
}

// helper functions for DP 

// contains - if slice contains an element of nodeid
func contains(slice []model.NodeId, element model.NodeId) bool {
	for _, v := range slice {
		if v == element {
			return true
		}
	}
	return false
}

// remove nil values from messages slice
func removeNilsMes(slice []*logics.Message) []*logics.Message {
	nonNilSlice := []*logics.Message{}
	for _, v := range slice {
		if v != nil {
			nonNilSlice = append(nonNilSlice, v)
		}
	}
	return nonNilSlice
}

func getSubsliceEncounters(encounters []model.Encounter, startTime, endTime float64) []model.Encounter {
	startIdx := sort.Search(len(encounters), func(i int) bool {
		return encounters[i].Time >= startTime
	})

	endIdx := sort.Search(len(encounters), func(i int) bool {
		return encounters[i].Time > endTime
	})

	return encounters[startIdx:endIdx]

}

func simulateBatchEncounters(config *model.Config, encounters []model.Encounter, rows *sql.Rows, sim_n_splits int, multiplier_timestep float64, time_step float32, startTime float64, endTime float64, expName string, EpochLoadChan chan *model.EpochLoad, messages map[model.NodeId][]*logics.Message, logic logics.Logic) {

	wasSplit := false //flag to know if sort is needed
	for rows.Next() {
		var enc model.Encounter
		if err := rows.Scan(&enc.DatasetName, &enc.Distance,
			&enc.ExperimentName, &enc.Time,
			&enc.Node1, &enc.Node2, &enc.Duration,
			&enc.X, &enc.Y, &enc.Z, &enc.PPBR); err != nil {
			log.Warnf("invalid encounter %v ", err)
			continue
		}

		// TODO: check this logic - why are we multiplying the time_step by a multipler here?
		// encs := splitEncounter(&enc, float64(sim_n_splits), float64(time_step))
		encs := splitEncounter(&enc, float64(sim_n_splits), float64(float32(multiplier_timestep)*time_step))

		if len(encs) > 1 {
			wasSplit = true
			for _, e := range encs {
				encounters = append(encounters, *e)
			}
		} else {
			encounters = append(encounters, *(encs[0]))
		}

	}
	log.Infof("finished splitting big encounters")


	//sort if needed
	if wasSplit {
		// Define a custom less function that compares Time, node1, and node2 fields.
		lessFunc := func(i, j int) bool {
			if encounters[i].Time != encounters[j].Time {
				return encounters[i].Time < encounters[j].Time
			}
			if encounters[i].Node1 != encounters[j].Node1 {
				return encounters[i].Node1 < encounters[j].Node1
			}
			return encounters[i].Node2 < encounters[j].Node2
		}

		// Sort the encounters slice using sort.Slice and the custom less function.
		sort.Slice(encounters, lessFunc)

	}
	log.Infof("finished sorting new encounters list")

	log.Infof("found %v encounters", len(encounters))

	// timestamp := 0.0
	// iterate all encounters

	//now, if the logic involves geolocation
	//like HumaNets/HumaNets-DP,
	//we need to manipulate the data a little bit
	//get only the encounters relevant the simulation

	encounters = getSubsliceEncounters(encounters, startTime, endTime)
	prev := encounters[0].Time

	// TODO: logic here is new, should compare with old version and logic check 
	for ind, encounter := range encounters {
		if encounter.Time > endTime { //finished this simulation,break
			log.Info("Stop! we fininshed running the experiment.")
			break
		}
		if encounter.Time < startTime { //skip this encounter, too soon
			continue
		}
		//in here, we check if the nodes in the encounter
		//are alive in the timestamp

		/*

		if strings.HasPrefix(logic.GetLogicName(), "addressing") || strings.HasPrefix(logic.GetLogicName(), "mirage") {
			//check if one of the nodes does not exists
			//if this is the case, move on 
			var TimesNode1 any
			var TimesNode2 any
			var ok, ok2 bool
			var t1, t2 float64
			if TimesNode1, ok = times_data.Load(encounter.Node1); !ok {
				//log.Infof("node1 %v does not appear in the timely data", encounter.Node1)

				continue
			}
			if TimesNode2, ok2 = times_data.Load(encounter.Node2); !ok2 {
				//log.Infof("node2 %v does not appear in the timely data", encounter.Node2)
				continue
			}
			t1 = TimesNode1.(float64)
			t2 = TimesNode2.(float64)

			//skip encounter if the nodes are not alive again
			if t1 > encounter.Time || t2 > encounter.Time {
				//log.Infof("%v or %v are still asleep at %v", encounter.Node1, encounter.Node2, encounter.Time)
				continue
			}

		}

		*/

		//time to calculate the epoch load
		if prev != encounter.Time {
			// calculate the load
			load, avg := logics.GetLoadandAvg()
			// calculate the epochs load and avg
			epl := model.EpochLoad{
				ExperimentName: expName,
				Now:            encounter.Time,
				Prev:           prev,
				Load:           load,
				AvgLoad:        avg,
			}
			prev = encounter.Time
			// //send to the epoch load channel
			EpochLoadChan <- &epl
		}
		//update encounter with experiment
		encounter.ExperimentName = expName
		//call a function that based on an encounter
		//and the messges slice, will simulate
		//the encounter
		log.Infof("Simulating encounter %v. %v encounters to go", ind, len(encounters)-(ind+1))
		encounterSim(config, &encounter, messages, logic)
	}
}