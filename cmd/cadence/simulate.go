package main

import (
	"database/sql"
	"errors"
	model "marathon-sim/datamodel"
	logics "marathon-sim/logic"
	"os"
	"os/exec"
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
var nodeTimeMutex sync.Mutex
var numEpochsToSimulate int

var encounterChan chan *model.Encounter
var bandChan chan *model.Bandwidths
var messageDBChan chan *model.MessageDB
var receivedmessageDBChan chan *model.DeliveredMessageDB

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

func getTimesFromDB_Orig(startTime, endTime float64, timeChan chan float64) {

	var rows *sql.Rows
	var err error

	log.Info("fetching times from database.  This can take awhile depending upon the size of the dataset.")
	if endTime > 0 {
		rows, err = model.DB.Table("events").Select("distinct time").Where("dataset_name=? and time>=? and time<=?", dataset.DatasetName, startTime, endTime).Order("time").Rows()
	} else {
		rows, err = model.DB.Table("events").Select("distinct time").Where("dataset_name=? and time>=?", dataset.DatasetName, startTime).Order("time").Rows()
	}
	if err != nil {
		log.Panicf("cannot query DB: %v", err)
	}

	allTimes := make([]float64, 0, 1024*1024*100) // allocate 100MB for this task
	for rows.Next() {
		var t float64
		if err := rows.Scan(&t); err != nil {
			log.Infof("invalid time: %v", err)
			continue
		}
		allTimes = append(allTimes, t)
	}
	rows.Close()
	numEpochsToSimulate = len(allTimes)
	log.Infof("finished grabbing times from DB; there are %v epochs to simulate", numEpochsToSimulate)
	for _, t := range allTimes {
		timeChan <- t
	}
	log.Info("no more times to simulate")
	close(timeChan)
}

func getTimesFromDB(startTime, endTime float64, timeChan chan float64, time_step float32) {

	allTimes := make([]float64, 0) // allocate 100MB for this task

	// Convert timeStep to float64 for consistency in calculations
	timeStep64 := float64(time_step)

	// Start the loop at the startTime and increment by timeStep until endTime is reached
	for i := startTime; i <= endTime; i += timeStep64 {
		allTimes = append(allTimes, i)
	}

	numEpochsToSimulate = len(allTimes)
	log.Infof("finished grabbing times; there are %v epochs to simulate", numEpochsToSimulate)
	for _, t := range allTimes {
		timeChan <- t
	}
	log.Info("no more times to simulate")
	close(timeChan)
}

// gets the first and last time this node was seen in the DB
func getNodeTimeBoundaries(id model.NodeId) (time1, time2 float64) {
	nodeTimeMutex.Lock()
	defer nodeTimeMutex.Unlock()

	rows, err := model.DB.Table("events").Select("min(time)").Where("dataset_name=? and node=?", dataset.DatasetName, id).Rows()
	if err != nil || !rows.Next() {
		log.Panicf("unable to find node %v in DB; err=%v", id, err)
	}
	rows.Scan(&time1)
	rows.Close()

	rows, err = model.DB.Table("events").Select("max(time)").Where("dataset_name=? and node=?", dataset.DatasetName, id).Rows()
	if err != nil || !rows.Next() {
		log.Panicf("unable to find node %v in DB; err=%v", id, err)
	}
	rows.Scan(&time2)
	rows.Close()
	return
}

// This is the function that actually does the hard work of a node.  It figures
// out its position and waits for all other workers to do the same.  It then
// figures out its distance to all other active nodes.
func nodeWorker(
	datasetName string, epochChan chan *Epoch,
	doneBarrier, positionsKnown *sync.WaitGroup, id model.NodeId,
	conditions model.EncounterConditions) {

	//starting a worker
	log.Infof("starting node worker for node %v", id)

	// first, figure out when this node first and last appears in the dataset
	firstAppeared, lastAppeared := getNodeTimeBoundaries(id) //(*times_data)[0].Time, (*times_data)[len(*times_data)-1].Time //times_data[0], times_data[1]
	//firstAppeared, lastAppeared := (*times_data)[0].Time, (*times_data)[len(*times_data)-1].Time //times_data[0], times_data[1]
	// log.Infof("node %v, start %v, end %v", id, firstAppeared, lastAppeared)
	// next, wait for epochs to come in fron the epochChan.  When a new epoch
	// arrives, indicating that there's a new epoch to consider, first compute
	// the start and end positions of this node for the epoch.  Then, wait for
	// all other nodes to do the same.  And finally, compute the distances to
	// each node to figure out if there are any encounters.
	var timeOfLastPositionUpdate float64
	//var err error

	// repeat for each new epoch; wait for epoch to begin by receiving on the
	// `epochChan`
	for epoch := range epochChan {
		var timeBounds *startEndEvents
		// if you first appeared after the epoch or last appeared before this
		// epoch even began, then this worker shouldn't bother doing anything
		if firstAppeared >= epoch.prev || lastAppeared <= epoch.now {
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
			//activeNodes.Store(id, nil) // not active during this epoch
			positionsKnown.Done()
			doneBarrier.Done()
			continue
		}
		// indicate that we're active during this epoch
		activeNodes.Store(id, timeBounds)

		// inform the node logic that we have a new position
		if timeOfLastPositionUpdate != timeBounds.start.Time {
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
		activeNodes.Range(func(k, v any) bool {

			// // if the other node isn't active, it'll set v to nil
			// if v == nil {
			// 	return true
			// }
			otherNodeId := k.(model.NodeId)
			// let's do these comparisons once, and let's not compare against
			// ourselves.  Arbitrarily, we'll say that nodes i and j will
			// compute distances iff i < j.
			if id >= otherNodeId {
				return true
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
				}
				//all conditions met! update the conditions if needed
				distance := 200.0
				for _, condition := range conditions {
					condition.UpdateEncounterConsequences(enc)
					if condition.GetType() == model.ConditionTypeDistanceInMeters {
						distance = condition.(*model.DistanceCondition).GetDistance()
					}
				}
				enc.Distance = float64(distance)

				//log the updated encounter
				CheckEncounter(enc, epoch)
				//encounterChan <- enc
			}
			return true
		})

		doneBarrier.Done() // signal that we're done
	}
}

func simulate(logicName, messagesFile string, conditionsFile string, startTime, endTime float64, datasetName, expName, investName string, max_buffer int, min_buffer int, min_message_size float32, max_message_size float32, sim_n_splits int, time_step float32, messageGeneratorOption string, messagesTemplatePath string, generatorScriptPath string, dbFile string) {
	var doneBarrier sync.WaitGroup
	var positionsKnown sync.WaitGroup
	var err error
	//get logic engine

	logic, err = logics.GetLogicByName(logicName)
	if err != nil {
		log.Fatalf("cannot retrieve logic engine with name %v: %v", logicName, err)
	}
	//init the delivered messages queue
	logics.DeliveredMessagesQueue = model.NewFastQueue()
	//set conditions
	conditions, conditionsDesciption, err := model.ReadConditionsFile(conditionsFile)
	if err != nil {
		log.Fatalf("cannot read conditions file (%v): %v", conditionsFile, err)
	}
	r := model.DB.First(&dataset, "dataset_name=?", datasetName)
	if r.Error != nil || r.RowsAffected != 1 {
		log.Fatalf("error retrieving dataset with name %v", datasetName)
	}
	experimentName = expName

	// create an experiment and save it in the DB
	exp := &model.Experiment{
		ExperimentName:     expName,
		DatasetName:        datasetName,
		Investigator:       investName,
		DistanceConditions: conditionsDesciption,
		CommandLine:        strings.Join(os.Args, ";"),
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

	log.Infof("beginning simulation on dataset %v; will run from %v to %v",
		datasetName, startTime, endTime)
	log.Infof("recording results to %v", expName)

	rows, err := model.DB.Table("events").Select("distinct node").Where("dataset_name=? and time>=? and time<=?", datasetName, startTime, endTime).Rows()
	if err != nil {
		log.Infof("error in fetching the nodes")
	}

	var nodes []model.NodeId
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

	// //rows, err := model.DB.Table("events").Where("dataset_name=? and time>=? and time<=?", datasetName, startTime, endTime).Order("time").Rows()
	// rows, err := model.DB.Table("events").Where("dataset_name=?", datasetName).Order("time").Rows()
	// if err != nil {
	// 	log.Infof("error in fetching the events for profiling")
	// }

	// //prepare the data
	// times_data, nodes := logic.DataPreparation(rows, nodes_list, log, minevents)
	//init the message status monitor
	model.MonitorInit()
	// allocate some stuff
	doneBarrier.Add(len(nodes))
	positionsKnown.Add(len(nodes))
	activeNodes = new(sync.Map)
	epochChans := make([]chan *Epoch, len(nodes))
	encounterChan = make(chan *model.Encounter, 1000)                  // buffer size of 128 is arbitrary
	messageDBChan = make(chan *model.MessageDB, 1000)                  // buffer size of 128 is arbitrary
	receivedmessageDBChan = make(chan *model.DeliveredMessageDB, 1000) // buffer size of 128 is arbitrary
	EpochLoadChan := make(chan *model.EpochLoad, 1000)
	bandChan = make(chan *model.Bandwidths, 1000) // channel for bandwidth documnetation
	prefetch = new(prefetcher)
	prefetch.init()
	//send the relevant channels to logics
	logics.AssignChannels(messageDBChan, receivedmessageDBChan)
	// kick off the encounter recorder as a separate goroutine.  this goroutine
	// will just wait for data (encounters) sent via the encounterChan, and then
	// will record them in the DB (in the Encounters table)
	var recorderSyncBarrier sync.WaitGroup
	var recorderSyncBarrierDB sync.WaitGroup
	var recorderSyncBarrierRecievedDB sync.WaitGroup
	var recorderSyncBarrierEpochLoadsDB sync.WaitGroup
	var recorderSyncBarrierBandwidth sync.WaitGroup

	//sign that encounters creation is complete
	//as it done through epoch
	//create the data to send to messages DB
	//this is the first part - before the encounters
	distance := 20.0 //default distance
	for _, condition := range conditions {
		if condition.GetType() == model.ConditionTypeDistanceInMeters {
			distance = condition.(*model.DistanceCondition).GetDistance()
		}

	}

	// Create a variable to hold the result
	var existingRow model.DatasetEncounterEnumeration
	var encountersExists bool
	// Check if a row with the specified values exists
	if err := model.DB.Where("distance = ? AND dataset_name = ? AND complete = ? and duration=?", distance, datasetName, true, time_step).First(&existingRow).Error; err != nil {
		// Handle the case where the row does not exist
		if errors.Is(err, gorm.ErrRecordNotFound) {
			encountersExists = false
		} else {
			log.Errorf("Error checking for an existing row: %v", err)
		}
	} else {
		// Handle the case where the row exists
		encountersExists = true

	}

	//encounters enumeration was not found
	if !encountersExists {
		log.Errorf("creating encounters")
		//channel for encounter records
		go model.RecordEncounters(experimentName, encounterChan, &recorderSyncBarrier)
		recorderSyncBarrier.Add(1)
		start_enc_calc := time.Now()
		// kick off each node worker
		for i, n := range nodes {
			//create the channel for epochs of the node
			epochChans[i] = make(chan *Epoch)
			go nodeWorker(
				datasetName,
				epochChans[i],
				&doneBarrier,
				&positionsKnown,
				n,
				conditions)

		}
		// fetch times from database
		timeChan := make(chan float64, 1024)
		go getTimesFromDB(startTime, endTime, timeChan, time_step)

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
		EmptyEncountnersManager()
		//update that we finished creating the encounters
		create_encounters := &model.DatasetEncounterEnumeration{
			Distance:    float32(distance),
			Duration:    time_step,
			DatasetName: datasetName,
			Complete:    true,
		}
		if r := model.DB.Create(create_encounters); r.Error != nil {
			log.Infof("cannot sign that we finished the encounter listing")
		}
		elapsed_enc := time.Since(start_enc_calc)
		log.Infof("Elapsed time from encounters creation start: %v", elapsed_enc.String())

		close(encounterChan)
		recorderSyncBarrier.Wait()
	}
	//message generation
	//first, check if we need to create it
	//if it exists, use it
	//elsewhere, create the file
	if info, err := os.Stat(messagesFile); err == nil && !info.IsDir() {
		log.Info("The message file exists. skip creation")
	} else {
		log.Info("The message file doesn't exists. create it")
		pythonFile := generatorScriptPath
		//the set of arguments for message generation script are:
		// #1-db string
		// #2-dataset name
		// #3-template file
		// #4-min message size
		// #5-max message size
		// #6-start time of experiment
		// #7-output file
		// #8- N^2 messages or 1-1 messages
		// #9 - end time of experiment
		// #10 - minimum distance between nodes
		cmd := exec.Command("python3", pythonFile, dbFile, datasetName, messagesTemplatePath,
			strconv.FormatFloat(float64(min_message_size), 'f', -1, 64),
			strconv.FormatFloat(float64(max_message_size), 'f', -1, 64),
			strconv.FormatFloat(startTime, 'f', -1, 64),
			messagesFile, messageGeneratorOption,
			strconv.FormatFloat(endTime, 'f', -1, 64), strconv.FormatFloat(float64(distance), 'f', -1, 64))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	//checking if there are nodes to remove from messaging
	//if os, do it
	var messages map[model.NodeId][]*logics.Message
	//load the messages
	messages, err = logics.LoadMessages(messagesFile, log, logic)
	if err != nil {
		log.Info(err)
		os.Exit(0)
	}
	//throughput calculation
	//first-gather the amount of messages
	mes_number := 0
	for n := range messages {
		mes_number += len(messages[n])
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
		logics.InitNodeMemory(min_buffer, max_buffer, n)
	}
	//for this case, we iterate the list of encounters and simulate the message transfer
	//get the encounters
	var encounters []model.Encounter
	rows, err = model.DB.Table("encounters").Select("*").Where("dataset_name=? and distance=?", datasetName, distance).Order("time ASC").Rows()
	if err != nil {
		log.Infof("error in fetching the events for profiling")
	}
	splited := false //flag to know if sort is needed
	for rows.Next() {
		var enc model.Encounter
		if err := rows.Scan(&enc.DatasetName, &enc.Distance,
			&enc.ExperimentName, &enc.Time,
			&enc.Node1, &enc.Node2, &enc.Duration,
			&enc.X, &enc.Y, &enc.Z); err != nil {
			log.Warnf("invalid encounter %v ", err)
			continue
		}
		encs := splitEncounter(&enc, float64(sim_n_splits), float64(time_step))
		if len(encs) > 1 {
			splited = true
			for _, e := range encs {
				encounters = append(encounters, *e)
			}
		} else {
			encounters = append(encounters, *(encs[0]))
		}

	}
	log.Infof("finished splitting big encounters")
	//sort if needed
	if splited {
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
	prev := encounters[0].Time
	for _, encounter := range encounters {
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
		encounterSim(&encounter, messages, logic)
	}

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
	latencies_from_table := model.DB.Table("delivered_message_dbs").Select("avg(deliver_time),avg(hops)").Where("experiment_name=?", expName).Row()
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
func setMessagesEncounterPerNode(messages map[model.NodeId][]*logics.Message, nodeid model.NodeId, encounter model.Encounter) {
	st_time := encounter.Time
	end_time := st_time + float64(encounter.Duration)
	//add the messages
	var tmp_slice_storage []*logics.Message
	for _, mes := range messages[nodeid] {
		//add messages from before the epoch,
		//or before its end
		if mes.CreationTime <= end_time {
			logics.OriginateMessage(nodeid, mes)

		} else {
			//add the message to a tmp slice
			//so it can be stored for the rest
			//of the simulation
			tmp_slice_storage = append(tmp_slice_storage, mes)
		}

	}
	//save only the messages that weren't added yet
	messages[nodeid] = tmp_slice_storage
}

// simulate the encounter
func encounterSim(encounter *model.Encounter, messages map[model.NodeId][]*logics.Message, logic logics.Logic) {
	node1 := encounter.Node1
	node2 := encounter.Node2
	//set the messages in the appropriate queues
	setMessagesEncounterPerNode(messages, node1, *encounter)
	setMessagesEncounterPerNode(messages, node2, *encounter)
	band := logic.HandleEncounter(encounter)
	// create the struct to store bandwidth
	b := &model.Bandwidths{Dataset: encounter.DatasetName,
		Distance:       encounter.Distance,
		Logic:          logic.GetLogicName(),
		Bandwidth:      band,
		Hash:           model.CalculateHash(encounter.Time, model.NodeIdInt(node1), model.NodeIdInt(node2)),
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
