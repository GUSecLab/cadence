package main

import (
    //"encoding/json"
    //"io"
    model "marathon-sim/datamodel"
    //"os"
    "testing"
    //"github.com/jinzhu/now"
    //logger "github.com/sirupsen/logrus"
)

//Set up Databse connection if you want to test didCrossPaths function with the data from DB
/*
func setupDB() {
    log = logger.New()
    l, _ := logger.ParseLevel("DEBUG")
    log.SetLevel(l)
    customFormatter := new(logger.TextFormatter)
    customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
    log.SetFormatter(UTCFormatter{customFormatter})
    customFormatter.FullTimestamp = true

    f, err := os.Open("config_avery_test.json")
    if err != nil {
        log.Fatal("Error reading test_config.json file:", err)
    }
    defer f.Close()
    b, err := io.ReadAll(f)
    if err != nil {
        log.Fatal("Error reading test_config.json file")
    }
    type DBConfig struct {
        DBType   string `json:"dbType"`
        DBString string `json:"dbString"`
    }
    var dbConfig DBConfig
    if err := json.Unmarshal(b, &dbConfig); err != nil {
        log.Fatal("Error parsing test_config.json file")
    }

    // initialize the DB
    model.Init(log, dbConfig.DBType, dbConfig.DBString)
}
*/

func TestDidCrossPathsGHCI(t *testing.T) {
    
    //setupDB()
    conditions,_ ,_  := model.ReadConditionsFile("conditions.json")

    /*The following tests utilized some examples from local. */

    //Test case 1: baseline- intersect within experiment

    t.Run("intersect within experiment", func(t *testing.T) {
        testEpoch_1 := Epoch{now:20.0, prev: 0.0, next: 30.0}
        n1EventStart_1 := model.Event{Time: 2.0, X: 0.0, Y: 0.0, Z: 0.0}
        n1EventEnd_1 := model.Event{Time: 12.0, X: 10.0, Y: 10.0, Z: 10.0}
        n1StartEndEvents_1 := startEndEvents{&n1EventStart_1, &n1EventEnd_1}
        n2EventStart_1 := model.Event{Time: 5.0, X: 10.0, Y: 0.0, Z: 10.0}
        n2EventEnd_1 := model.Event{Time: 15.0, X: 0.0, Y: 10.0, Z: 10.0}
        n2StartEndEvents_1 := startEndEvents{&n2EventStart_1, &n2EventEnd_1}
        

        allConditionsPassed_1, timeOfClosestEncounter_1, _, _, _ := didCrossPaths(&testEpoch_1, &n1StartEndEvents_1,&n2StartEndEvents_1, conditions, nil, nil)
        expectedCross_1 := true
        if allConditionsPassed_1 != expectedCross_1{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_1, timeOfClosestEncounter_1)
        }
    })

    //Test case 2: baseline- no intersection

    t.Run("no intersection", func(t *testing.T) {

        testEpoch_2 := Epoch{now:12.0, prev: 6.0, next: 30.0}
        n1EventStart_2 := model.Event{Time: 2.0, X: 0.0, Y: 0.0, Z: 0.0}
        n1EventEnd_2 := model.Event{Time: 6.0, X: 5.0, Y: 5.0, Z: 5.0}
        n1StartEndEvents_2 := startEndEvents{&n1EventStart_2, &n1EventEnd_2}
        n2EventStart_2 := model.Event{Time: 6.0, X: 30.0, Y: 30.0, Z: 30.0}
        n2EventEnd_2 := model.Event{Time: 10.0, X: 35.0, Y: 35.0, Z: 35.0}
        n2StartEndEvents_2 := startEndEvents{&n2EventStart_2, &n2EventEnd_2}

        allConditionsPassed_2, timeOfClosestEncounter_2, _, _, _ := didCrossPaths(&testEpoch_2, &n1StartEndEvents_2,&n2StartEndEvents_2, conditions, nil, nil)
        expectedCross_2 := false
        if allConditionsPassed_2 != expectedCross_2{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_2, timeOfClosestEncounter_2)
        }
    })

    //Test case 3: baseline- intersect before experiment

    t.Run("intersect before experiment", func(t *testing.T) {

        testEpoch_3 := Epoch{now:14.0, prev: 11.0, next: 17.0}
        n1EventStart_3 := model.Event{Time: 10.0, X: 52.0, Y: 27.0, Z: 27.0}
        n1EventEnd_3 := model.Event{Time: 13.0, X: 62.0, Y: 37.0, Z: 37.0}
        n1StartEndEvents_3 := startEndEvents{&n1EventStart_3, &n1EventEnd_3}
        n2EventStart_3 := model.Event{Time: 10.0, X: 27.0, Y: 27.0, Z: 27.0}
        n2EventEnd_3 := model.Event{Time: 13.0, X: 28.0, Y: 28.0, Z: 28.0}
        n2StartEndEvents_3 := startEndEvents{&n2EventStart_3, &n2EventEnd_3}

        allConditionsPassed_3, timeOfClosestEncounter_3, _, _, _ := didCrossPaths(&testEpoch_3, &n1StartEndEvents_3,&n2StartEndEvents_3, conditions, nil, nil)
        expectedCross_3 := false
        if allConditionsPassed_3 != expectedCross_3{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_3, timeOfClosestEncounter_3)
        }
    })

    // Test case 4: baseline- intersect after experiment

    t.Run("intersect after experiment", func(t *testing.T) {

        testEpoch_4 := Epoch{now:12.0, prev: 11.0, next: 17.0}
        n1EventStart_4 := model.Event{Time: 10.0, X: 62.0, Y: 37.0, Z: 37.0}
        n1EventEnd_4 := model.Event{Time: 13.0, X: 50.0, Y: 27.0, Z: 27.0}
        n1StartEndEvents_4 := startEndEvents{&n1EventStart_4, &n1EventEnd_4}
        n2EventStart_4 := model.Event{Time: 10.0, X: 24.0, Y: 24.0, Z: 24.0}
        n2EventEnd_4 := model.Event{Time: 13.0, X: 27.0, Y: 27.0, Z: 27.0}
        n2StartEndEvents_4 := startEndEvents{&n2EventStart_4, &n2EventEnd_4}

        allConditionsPassed_4, timeOfClosestEncounter_4, _, _, _ := didCrossPaths(&testEpoch_4, &n1StartEndEvents_4,&n2StartEndEvents_4, conditions, nil, nil)
        expectedCross_4 := false
        if allConditionsPassed_4 != expectedCross_4{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_4, timeOfClosestEncounter_4)
        }

    })

    //Test case 5: edge- identical Start and End Points

    t.Run("identical start/end points", func(t *testing.T) {


        testEpoch_5 := Epoch{now:12.0, prev: 10.0, next: 17.0}
        n1EventStart_5 := model.Event{Time: 11.0, X: 0.0, Y: 0.0, Z: 0.0}
        n1EventEnd_5 := model.Event{Time: 13.0, X: -5.0, Y: -5.0, Z: -5.0}
        n1StartEndEvents_5 := startEndEvents{&n1EventStart_5, &n1EventEnd_5}
        n2EventStart_5 := model.Event{Time: 11.0, X: 0.0, Y: 0.0, Z: 0.0}
        n2EventEnd_5 := model.Event{Time: 13.0, X: -5.0, Y: -5.0, Z: -5.0}
        n2StartEndEvents_5 := startEndEvents{&n2EventStart_5, &n2EventEnd_5}

        allConditionsPassed_5, timeOfClosestEncounter_5, _, _, _ := didCrossPaths(&testEpoch_5, &n1StartEndEvents_5,&n2StartEndEvents_5, conditions, nil, nil)
        expectedCross_5 := true
        if allConditionsPassed_5 != expectedCross_5{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_5, timeOfClosestEncounter_5)
        }

    })

    //Test case 6: edge- parallel paths

    t.Run("parrallel paths", func(t *testing.T) {


        testEpoch_6 := Epoch{now:14.0, prev: 12.0, next: 17.0}
        n1EventStart_6 := model.Event{Time: 11.0, X: 0.0, Y: 0.0, Z: 0.0}
        n1EventEnd_6 := model.Event{Time: 13.0, X: -5.0, Y: -5.0, Z: -5.0}
        n1StartEndEvents_6 := startEndEvents{&n1EventStart_6, &n1EventEnd_6}
        n2EventStart_6 := model.Event{Time: 11.0, X: -5.0, Y: -5.0, Z: -5.0}
        n2EventEnd_6 := model.Event{Time: 13.0, X: -10.0, Y: -10.0, Z: -10.0}
        n2StartEndEvents_6 := startEndEvents{&n2EventStart_6, &n2EventEnd_6}

        allConditionsPassed_6, timeOfClosestEncounter_6, _, _, _ := didCrossPaths(&testEpoch_6, &n1StartEndEvents_6,&n2StartEndEvents_6, conditions, nil, nil)
        expectedCross_6 := true
        if allConditionsPassed_6 != expectedCross_6{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_6, timeOfClosestEncounter_6)
        }
    })

    //Test case 7: edge- one node stationary

    t.Run("one stationary node", func(t *testing.T) {

        testEpoch_7 := Epoch{now:14.0, prev: 11.0, next: 17.0}
        n1EventStart_7 := model.Event{Time: 11.0, X: 0.0, Y: 0.0, Z: 0.0}
        n1EventEnd_7 := model.Event{Time: 13.0, X: 0.0, Y: 0.0, Z: 0.0}
        n1StartEndEvents_7 := startEndEvents{&n1EventStart_7, &n1EventEnd_7}
        n2EventStart_7 := model.Event{Time: 11.0, X: 0.0, Y: 25.0, Z: 0.0}
        n2EventEnd_7 := model.Event{Time: 13.0, X: 0.0, Y: 0.0, Z: 0.0}
        n2StartEndEvents_7 := startEndEvents{&n2EventStart_7, &n2EventEnd_7}

        allConditionsPassed_7, timeOfClosestEncounter_7, _, _, _ := didCrossPaths(&testEpoch_7, &n1StartEndEvents_7,&n2StartEndEvents_7, conditions, nil, nil)
        expectedCross_7 := true
        if allConditionsPassed_7 != expectedCross_7{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_7, timeOfClosestEncounter_7)
        }
    })

    //Test case 8: edge- non-overlapping time

    t.Run("non overlapping time", func(t *testing.T) {

        testEpoch_8 := Epoch{now:10.0, prev: 0.0, next: 17.0}
        n1EventStart_8 := model.Event{Time: 0.0, X: 0.0, Y: 0.0, Z: 0.0}
        n1EventEnd_8 := model.Event{Time: 5.0, X: 25.0, Y: 25.0, Z: 25.0}
        n1StartEndEvents_8 := startEndEvents{&n1EventStart_8, &n1EventEnd_8}
        n2EventStart_8 := model.Event{Time: 6.0, X: 0.0, Y: 10.0, Z: 10.0}
        n2EventEnd_8 := model.Event{Time: 10.0, X: 25.0, Y: 0.0, Z: 0.0}
        n2StartEndEvents_8 := startEndEvents{&n2EventStart_8, &n2EventEnd_8}

        allConditionsPassed_8, timeOfClosestEncounter_8, _, _, _ := didCrossPaths(&testEpoch_8, &n1StartEndEvents_8,&n2StartEndEvents_8, conditions, nil, nil)
        expectedCross_8 := false
        if allConditionsPassed_8 != expectedCross_8{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_8, timeOfClosestEncounter_8)
        }
    })

    //Test case 9: both nodes on negative coordination

    t.Run("negative coordination", func(t *testing.T) {

        testEpoch_9 := Epoch{now:10.0, prev: 0.0, next: 17.0}
        n1EventStart_9 := model.Event{Time: 0.0, X: -1.0, Y: -1.0, Z: -1.0}
        n1EventEnd_9 := model.Event{Time: 10.0, X: -25.0, Y: -25.0, Z: -25.0}
        n1StartEndEvents_9 := startEndEvents{&n1EventStart_9, &n1EventEnd_9}
        n2EventStart_9 := model.Event{Time: 6.0, X: 0.0, Y: -10.0, Z: -10.0}
        n2EventEnd_9 := model.Event{Time: 10.0, X: -25.0, Y: -25.0, Z: -24.0}
        n2StartEndEvents_9 := startEndEvents{&n2EventStart_9, &n2EventEnd_9}

        allConditionsPassed_9, timeOfClosestEncounter_9, _, _, _ := didCrossPaths(&testEpoch_9, &n1StartEndEvents_9,&n2StartEndEvents_9, conditions, nil, nil)
        expectedCross_9 := true
        if allConditionsPassed_9 != expectedCross_9{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_9, timeOfClosestEncounter_9)
        }
    })

     //Test case 10: 

     t.Run("test case 10", func(t *testing.T) {

        testEpoch_10 := Epoch{now:8.0, prev: 0.0, next: 17.0}
        n1EventStart_10 := model.Event{Time: 0.0, X: -1.0, Y: -1.0, Z: -1.0}
        n1EventEnd_10 := model.Event{Time: 6.0, X: -25.0, Y: -25.0, Z: -25.0}
        n1StartEndEvents_10 := startEndEvents{&n1EventStart_10, &n1EventEnd_10}
        n2EventStart_10 := model.Event{Time: 6.0, X: 0.0, Y: -10.0, Z: -10.0}
        n2EventEnd_10 := model.Event{Time: 10.0, X: -25.0, Y: -25.0, Z: -24.0}
        n2StartEndEvents_10 := startEndEvents{&n2EventStart_10, &n2EventEnd_10}
    
        allConditionsPassed_10, timeOfClosestEncounter_10, _, _, _ := didCrossPaths(&testEpoch_10, &n1StartEndEvents_10,&n2StartEndEvents_10, conditions, nil, nil)
        expectedCross_10 := false
        if allConditionsPassed_10 != expectedCross_10{
            t.Errorf("Expected intersect to be %v, got %v for staggered times intersecting within the epoch", expectedCross_10, timeOfClosestEncounter_10)
        }
    })
}
