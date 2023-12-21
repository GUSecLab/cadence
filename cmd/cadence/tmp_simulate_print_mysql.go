package main

// import (
// 	"database/sql"
// 	model "marathon-sim/datamodel"
// 	logics "marathon-sim/logic"
// 	"os"
// 	"os/exec"
// 	"strconv"

// 	"sync"
// )

// var logic logics.Logic
// var dataset model.Dataset
// var experimentName string
// var nodeTimeMutex sync.Mutex
// var numEpochsToSimulate int

// var encounterChan chan *model.Encounter
// var bandChan chan *model.Bandwidths
// var messageDBChan chan *model.MessageDB
// var receivedmessageDBChan chan *model.DeliveredMessageDB

// var prefetch *prefetcher

// // this is a map where the key is the nodeid and the value is the start and end
// // events for the current epoch
// var activeNodes *sync.Map

// type Epoch struct {
// 	now  float64
// 	prev float64
// 	next float64
// }
// type startEndEvents struct {
// 	start *model.Event
// 	end   *model.Event
// }

// func getTimesFromDB_Orig(startTime, endTime float64, timeChan chan float64) {

// 	var rows *sql.Rows
// 	var err error

// 	log.Info("fetching times from database.  This can take awhile depending upon the size of the dataset.")
// 	if endTime > 0 {
// 		rows, err = model.DB.Table("events").Select("distinct time").Where("dataset_name=? and time>=? and time<=?", dataset.DatasetName, startTime, endTime).Order("time").Rows()
// 	} else {
// 		rows, err = model.DB.Table("events").Select("distinct time").Where("dataset_name=? and time>=?", dataset.DatasetName, startTime).Order("time").Rows()
// 	}
// 	if err != nil {
// 		log.Panicf("cannot query DB: %v", err)
// 	}

// 	allTimes := make([]float64, 0, 1024*1024*100) // allocate 100MB for this task
// 	for rows.Next() {
// 		var t float64
// 		if err := rows.Scan(&t); err != nil {
// 			log.Infof("invalid time: %v", err)
// 			continue
// 		}
// 		allTimes = append(allTimes, t)
// 	}
// 	rows.Close()
// 	numEpochsToSimulate = len(allTimes)
// 	log.Infof("finished grabbing times from DB; there are %v epochs to simulate", numEpochsToSimulate)
// 	for _, t := range allTimes {
// 		timeChan <- t
// 	}
// 	log.Info("no more times to simulate")
// 	close(timeChan)
// }

// func getTimesFromDB(startTime, endTime float64, timeChan chan float64, time_step float32) {

// 	allTimes := make([]float64, 0) // allocate 100MB for this task

// 	// Convert timeStep to float64 for consistency in calculations
// 	timeStep64 := float64(time_step)

// 	// Start the loop at the startTime and increment by timeStep until endTime is reached
// 	for i := startTime; i <= endTime; i += timeStep64 {
// 		allTimes = append(allTimes, i)
// 	}

// 	numEpochsToSimulate = len(allTimes)
// 	log.Infof("finished grabbing times; there are %v epochs to simulate", numEpochsToSimulate)
// 	for _, t := range allTimes {
// 		timeChan <- t
// 	}
// 	log.Info("no more times to simulate")
// 	close(timeChan)
// }

// // gets the first and last time this node was seen in the DB
// func getNodeTimeBoundaries(id model.NodeId) (time1, time2 float64) {
// 	nodeTimeMutex.Lock()
// 	defer nodeTimeMutex.Unlock()

// 	rows, err := model.DB.Table("events").Select("min(time)").Where("dataset_name=? and node=?", dataset.DatasetName, id).Rows()
// 	if err != nil || !rows.Next() {
// 		log.Panicf("unable to find node %v in DB; err=%v", id, err)
// 	}
// 	rows.Scan(&time1)
// 	rows.Close()

// 	rows, err = model.DB.Table("events").Select("max(time)").Where("dataset_name=? and node=?", dataset.DatasetName, id).Rows()
// 	if err != nil || !rows.Next() {
// 		log.Panicf("unable to find node %v in DB; err=%v", id, err)
// 	}
// 	rows.Scan(&time2)
// 	rows.Close()
// 	return
// }

// // This is the function that actually does the hard work of a node.  It figures
// // out its position and waits for all other workers to do the same.  It then
// // figures out its distance to all other active nodes.
// func nodeWorker(
// 	datasetName string, epochChan chan *Epoch,
// 	doneBarrier, positionsKnown *sync.WaitGroup, id model.NodeId,
// 	conditions model.EncounterConditions) {

// 	//starting a worker
// 	log.Infof("starting node worker for node %v", id)

// 	// first, figure out when this node first and last appears in the dataset
// 	firstAppeared, lastAppeared := getNodeTimeBoundaries(id) //(*times_data)[0].Time, (*times_data)[len(*times_data)-1].Time //times_data[0], times_data[1]
// 	//firstAppeared, lastAppeared := (*times_data)[0].Time, (*times_data)[len(*times_data)-1].Time //times_data[0], times_data[1]
// 	// log.Infof("node %v, start %v, end %v", id, firstAppeared, lastAppeared)
// 	// next, wait for epochs to come in fron the epochChan.  When a new epoch
// 	// arrives, indicating that there's a new epoch to consider, first compute
// 	// the start and end positions of this node for the epoch.  Then, wait for
// 	// all other nodes to do the same.  And finally, compute the distances to
// 	// each node to figure out if there are any encounters.
// 	var timeOfLastPositionUpdate float64
// 	//var err error

// 	// repeat for each new epoch; wait for epoch to begin by receiving on the
// 	// `epochChan`
// 	for epoch := range epochChan {
// 		var timeBounds *startEndEvents
// 		// if you first appeared after the epoch or last appeared before this
// 		// epoch even began, then this worker shouldn't bother doing anything
// 		if firstAppeared >= epoch.prev || lastAppeared <= epoch.now {
// 			//log.Debugf("node %v has no position at epoch %v because its times are %v to %v", id, *epoch, firstAppeared, lastAppeared)
// 			// activeNodes.Store(id, nil) // not active during this epoch
// 			positionsKnown.Done()
// 			doneBarrier.Done()
// 			continue
// 		}
// 		// otherwise, find the position of this node at the start and end of
// 		// the epoch
// 		timeBounds, _ = prefetch.getNodePositions(epoch, id)
// 		if timeBounds == nil {
// 			//log.Debugf("node %v has no position at epoch %v because it has no timebounds", id, *epoch)
// 			//activeNodes.Store(id, nil) // not active during this epoch
// 			positionsKnown.Done()
// 			doneBarrier.Done()
// 			continue
// 		}
// 		// indicate that we're active during this epoch
// 		activeNodes.Store(id, timeBounds)

// 		// inform the node logic that we have a new position
// 		if timeOfLastPositionUpdate != timeBounds.start.Time {
// 			logic.NewPositionCallback(id, dataset.CoordType, timeBounds.start.MarshalledLocation)
// 			timeOfLastPositionUpdate = timeBounds.start.Time
// 		}

// 		// effectively, tell other workers that we're done figuring out our
// 		// position
// 		positionsKnown.Done()

// 		// next, wait for all nodes to get their positions
// 		positionsKnown.Wait()

// 		//log.Debug("I think everyone else knows their positions")

// 		// compute distances to other nodes. Note that the Range function
// 		// executes the function on every element in the map.
// 		activeNodes.Range(func(k, v any) bool {

// 			// // if the other node isn't active, it'll set v to nil
// 			// if v == nil {
// 			// 	return true
// 			// }
// 			otherNodeId := k.(model.NodeId)
// 			// let's do these comparisons once, and let's not compare against
// 			// ourselves.  Arbitrarily, we'll say that nodes i and j will
// 			// compute distances iff i < j.
// 			if id >= otherNodeId {
// 				return true
// 			}

// 			otherNodeTimeBounds := v.(*startEndEvents)

// 			//log.Infof("computing closest distance between nodes %v and %v in this epoch", id, otherNodeId)
// 			crossedPaths, t, avX, avY, avZ := didCrossPaths(epoch, timeBounds, otherNodeTimeBounds, conditions,
// 				nil, nil)

// 			// if we crossed paths, let's inform the RecordEncounters goroutine
// 			// to save it to the DB.  We do this by sending the encounter to the
// 			// encounterChan.
// 			if crossedPaths {
// 				//create an encounter
// 				enc := &model.Encounter{
// 					DatasetName:    datasetName,
// 					ExperimentName: experimentName,
// 					Time:           t,
// 					Node1:          id,
// 					Node2:          otherNodeId,
// 					X:              avX,
// 					Y:              avY,
// 					Z:              avZ,
// 				}
// 				//all conditions met! update the conditions if needed
// 				distance := 200.0
// 				for _, condition := range conditions {
// 					condition.UpdateEncounterConsequences(enc)
// 					if condition.GetType() == model.ConditionTypeDistanceInMeters {
// 						distance = condition.(*model.DistanceCondition).GetDistance()
// 					}
// 				}
// 				enc.Distance = float64(distance)

// 				//log the updated encounter
// 				CheckEncounter(enc, epoch)
// 				//encounterChan <- enc
// 			}
// 			return true
// 		})

// 		doneBarrier.Done() // signal that we're done
// 	}
// }

// func simulate2(logicName, messagesFile string, conditionsFile string, startTime, endTime float64, datasetName, expName, investName string, max_buffer int, min_buffer int, min_message_size float32, max_message_size float32, sim_n_splits int, time_step float32, messageGeneratorOption string, messagesTemplatePath string, generatorScriptPath string, dbFile string) {
// 	var err error
// 	//get logic engine

// 	logic, err = logics.GetLogicByName(logicName)
// 	if err != nil {
// 		log.Fatalf("cannot retrieve logic engine with name %v: %v", logicName, err)
// 	}

// 	//set conditions
// 	conditions, _, err := model.ReadConditionsFile(conditionsFile)
// 	if err != nil {
// 		log.Fatalf("cannot read conditions file (%v): %v", conditionsFile, err)
// 	}

// 	//sign that encounters creation is complete
// 	//as it done through epoch
// 	//create the data to send to messages DB
// 	//this is the first part - before the encounters
// 	distance := 20.0 //default distance
// 	for _, condition := range conditions {
// 		if condition.GetType() == model.ConditionTypeDistanceInMeters {
// 			distance = condition.(*model.DistanceCondition).GetDistance()
// 		}

// 	}

// 	pythonFile := "tmp_script_for_output.py"
// 	//the set of arguments for message generation script are:
// 	// #1-db string
// 	// #2-dataset name
// 	// #3-template file
// 	// #4-min message size
// 	// #5-max message size
// 	// #6-start time of experiment
// 	// #7-output file
// 	// #8- N^2 messages or 1-1 messages
// 	// #9 - end time of experiment
// 	// #10 - minimum distance between nodes
// 	cmd := exec.Command("python3", pythonFile, dbFile, datasetName, messagesTemplatePath,
// 		strconv.FormatFloat(float64(min_message_size), 'f', -1, 64),
// 		strconv.FormatFloat(float64(max_message_size), 'f', -1, 64),
// 		strconv.FormatFloat(startTime, 'f', -1, 64),
// 		messagesFile, messageGeneratorOption,
// 		strconv.FormatFloat(endTime, 'f', -1, 64), strconv.FormatFloat(float64(distance), 'f', -1, 64))
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	err = cmd.Run()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	os.Exit(0)
// }
