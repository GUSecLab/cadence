package main

import (
	"encoding/json"
	"io"
	model "marathon-sim/datamodel"
	"os"
	"testing"
	"sync"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"time"

)


func setup() {

	log = logger.New()
	l, _ := logger.ParseLevel("DEBUG")
	log.SetLevel(l)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	log.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	f, err := os.Open("configs/unit_tests/config_simulation_test.json")
	if err != nil {
		log.Fatal("Error reading test_config.json file:", err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatal("Error reading test_config.json file")
	}
	var dbConfig *model.Config
	if err := json.Unmarshal(b, &dbConfig); err != nil {
		log.Fatal("Error parsing test_config.json file")
	}
	
	// initialize the DB
	model.Init(log, dbConfig)
}



// These test cases works for the imported database through config_avery_simulation. 
// Uncomment the test cases at the bottom if you used database of marathondev. 

// TestGetRelevantNodes is a unit test function that tests the getRelevantNodes function.

// this test might take a little bit of time, it may fail due to timeout so make sure to give
// it at least 60 seconds. 

func TestGetRelevantNodes(t *testing.T) {
	setup()
	datasetName := "cabspotting"
	startTime := 1211018404.0
	endTime := 1213089934.0

	// Test case 1: ktop_nodes > 1
	t.Run("KTopNodesGreaterThanOne", func(t *testing.T) {
		ktopNodes := 5

		//Tests used by Avery
		expectedNodes := []model.NodeId{516, 492, 207, 192, 173}
		expectedNodeMap := make(map[model.NodeId]bool)
		for _, node := range expectedNodes {
			expectedNodeMap[node] = true
		}

		nodes, nodeMap := getRelevantNodes(ktopNodes, datasetName, startTime, endTime)
		for _, n := range expectedNodes {
			assert.Contains(t, nodes, n)
			assert.True(t, nodeMap[n])
		}

	})

	// Test case 2: ktop_nodes <= 1
	t.Run("KTopNodesLessThanOrEqualToOne", func(t *testing.T) {
		ktopNodes := 1
		nodes, _ := getRelevantNodes(ktopNodes, datasetName, startTime, endTime)
		// assert.Len(t, nodes, 535)

		// Test used by Avery
		assert.Len(t, nodes, 536)
	})
}


// TestGetNodeTimeBoundaries is a unit test function that tests the getNodeTimeBoundaries function.
func TestGetNodeTimeBoundaries(t *testing.T) {
	setup()
	datasetName := "cabspotting"
	r := model.DB.First(&dataset, "dataset_name=?", datasetName)
	assert.Nil(t, r.Error, "error in design of test case")
	assert.Equal(t, r.RowsAffected, int64(1), "error in design of test case")
	/*Tests used by Avery*/
	// Test case 1
	t.Run("node5", func(t *testing.T) {
		t1, t2 := getNodeTimeBoundaries(6)
		assert.EqualValues(t, 1211018458, t1)
		assert.EqualValues(t, 1213034473, t2)
	})

	// Test case 2
	t.Run("node15", func(t *testing.T) {
		t1, t2 := getNodeTimeBoundaries(16)
		assert.EqualValues(t, 1211026811, t1)
		assert.EqualValues(t, 1213034469, t2)
	})
}
/* Uncomment this if you used marathondev database
// TestGetRelevantNodes is a unit test function that tests the getRelevantNodes function.
func TestGetRelevantNodes(t *testing.T) {
	setup()
	datasetName := "cabspotting"
	startTime := 1211018404.0
	endTime := 1213089934.0

	// Test case 1: ktop_nodes > 1
	t.Run("KTopNodesGreaterThanOne", func(t *testing.T) {
		ktopNodes := 5

		expectedNodes := []model.NodeId{515, 491, 206, 191, 172}
		expectedNodeMap := make(map[model.NodeId]bool)
		for _, node := range expectedNodes {
			expectedNodeMap[node] = true
		}

		nodes, nodeMap := getRelevantNodes(ktopNodes, datasetName, startTime, endTime)
		for _, n := range expectedNodes {
			assert.Contains(t, nodes, n)
			assert.True(t, nodeMap[n])
		}

	})

	// Test case 2: ktop_nodes <= 1
	t.Run("KTopNodesLessThanOrEqualToOne", func(t *testing.T) {
		ktopNodes := 1

		nodes, _ := getRelevantNodes(ktopNodes, datasetName, startTime, endTime)
		assert.Len(t, nodes, 535)
	})
}

// TestGetNodeTimeBoundaries is a unit test function that tests the getNodeTimeBoundaries function.
func TestGetNodeTimeBoundaries(t *testing.T) {
	setup()
	datasetName := "cabspotting"
	r := model.DB.First(&dataset, "dataset_name=?", datasetName)
	assert.Nil(t, r.Error, "error in design of test case")
	assert.Equal(t, r.RowsAffected, int64(1), "error in design of test case")

	// Test case 1
	t.Run("node5", func(t *testing.T) {
		t1, t2 := getNodeTimeBoundaries(5)
		assert.EqualValues(t, 1211018458, t1)
		assert.EqualValues(t, 1213034473, t2)
	})

	// Test case 2
	t.Run("node15", func(t *testing.T) {
		t1, t2 := getNodeTimeBoundaries(15)
		assert.EqualValues(t, 1211026811, t1)
		assert.EqualValues(t, 1213034469, t2)
	})
}
*/

// This is unit test for getTimesFromDB.
func TestGetTimesFromDB (t *testing.T){
	t.Run("TestTimeChan1", func(t *testing.T) {
		log = logger.New()
		l, _ := logger.ParseLevel("DEBUG")
		log.SetLevel(l)
		customFormatter := new(logger.TextFormatter)
		customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
		log.SetFormatter(UTCFormatter{customFormatter})
		customFormatter.FullTimestamp = true

		resultTimeChan := make(chan float64, 128)
		startTime := 2.6
		endTime := 10.8
		expectedTimeList := []float64{2.6, 4.6, 6.6, 8.6, 10.6}

		getTimesFromDB(startTime, endTime, resultTimeChan, 2.0)
		idx := 0
		for time := range resultTimeChan{
			assert.EqualValues(t, expectedTimeList[idx], time)
			idx++
		}
	})
}

// This is unit test for populateEpochChan

// PopulateEpochChans logic is correct, the waitgroup logic is correct. 
// Just need to fix the last epoch error
func TestPopulateEpochChans (t *testing.T) {
	//set up logger
    log = logger.New()
    l, _ := logger.ParseLevel("DEBUG")
    log.SetLevel(l)
    customFormatter := new(logger.TextFormatter)
    customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
    log.SetFormatter(UTCFormatter{customFormatter})
    customFormatter.FullTimestamp = true

    //prep the parameters
    timeChan := make(chan float64, 128)
	timeStep := float32(2.0)
    getTimesFromDB(2.6, 10.8, timeChan, timeStep)
    nodes := []model.NodeId{0, 1, 2}

    var doneBarrier sync.WaitGroup
	var positionsKnown sync.WaitGroup
    doneBarrier.Add(len(nodes))
    positionsKnown.Add(len(nodes))
    epochChans := make([]chan *Epoch, len(nodes))
    for i := range epochChans {
		// Be careful to handle the epoch channel's intialization. 
        epochChans[i] = make(chan *Epoch,3)
    }

    reporterChan := make (chan float64)
    expectedList := []Epoch {{prev : 2.6, now: 4.6, next: 6.6,}, 
                            {prev : 2.6, now: 4.6, next: 6.6,},
                            {prev : 2.6, now: 4.6, next: 6.6,}}
	reportTime := 6.6
    
    go populateEpochChans(nodes, timeChan, &doneBarrier, &positionsKnown, epochChans, reporterChan)
	
	/*
	doneBarrier.Done()
	doneBarrier.Done()
	doneBarrier.Done()
	epoch := <- epochChans[1]
	report := <-reporterChan
	assert.EqualValues(t, expectedList[1], *epoch)
	assert.EqualValues(t, 8.6, report)
	epoch = <-epochChans[2]
	assert.EqualValues(t, expectedList[2], *epoch)

	expectedList[1].prev +=float64(2.0)
    expectedList[1].now +=float64(2.0)
    expectedList[1].next +=float64(2.0)
	
	doneBarrier.Done()
	doneBarrier.Done()
	doneBarrier.Done()
	epoch = <- epochChans[1]
	assert.EqualValues(t, expectedList[1], *epoch)

	report = <-reporterChan
	assert.EqualValues(t, 10.6, report)
	//epoch = <- epochChans[1]
	//assert.EqualValues(t, expectedList[1], *epoch)

	expectedList[1].prev +=float64(2.0)
    expectedList[1].now +=float64(2.0)
    expectedList[1].next +=float64(2.0)
	
	doneBarrier.Done()
	doneBarrier.Done()
	doneBarrier.Done()
	epoch = <- epochChans[1]
	assert.EqualValues(t, expectedList[1], *epoch)

	report = <-reporterChan
	assert.EqualValues(t, 12.6, report)
	*/

    // iterate over Epochs	
    for i := 0; i<len(timeChan)-3; i++ {
		for j := 0; j < len(expectedList); j++{
			doneBarrier.Done()
			epoch := <- epochChans[j]
			assert.EqualValues(t, expectedList[j], *epoch)
			expectedList[j].prev +=float64(timeStep)
    		expectedList[j].now +=float64(timeStep)
    		expectedList[j].next +=float64(timeStep)
		}
		report := <-reporterChan
		assert.EqualValues(t, reportTime, report)
		reportTime += float64(timeStep)
	}
}

func TestSimSppedReporter (t *testing.T) {
	//set up logger
    log = logger.New()
    l, _ := logger.ParseLevel("DEBUG")
    log.SetLevel(l)
    customFormatter := new(logger.TextFormatter)
    customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
    log.SetFormatter(UTCFormatter{customFormatter})
    customFormatter.FullTimestamp = true

    //prep the parameters
    timeChan := make(chan float64, 128)
	timeStep := float32(2.0)
    getTimesFromDB(2.6, 10.8, timeChan, timeStep)
    nodes := []model.NodeId{0, 1, 2}

    var doneBarrier sync.WaitGroup
	var positionsKnown sync.WaitGroup
    doneBarrier.Add(len(nodes))
    positionsKnown.Add(len(nodes))
    epochChans := make([]chan *Epoch, len(nodes))
    for i := range epochChans {
		// Be careful to handle the epoch channel's intialization. 
        epochChans[i] = make(chan *Epoch,3)
    }

    reporterChan := make (chan float64)
	go simSpeedReporter(reporterChan)
	go populateEpochChans(nodes, timeChan, &doneBarrier, &positionsKnown, epochChans, reporterChan)
	for i := 0; i<len(timeChan)-3; i++ {
		for j := 0; j < 3; j++{
			time.Sleep(1*time.Second)
			doneBarrier.Done()
		}
	}
}

