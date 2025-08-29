package main

import (
	"encoding/json"
	"fmt"
	model "marathon-sim/datamodel"
	"marathon-sim/lens"
	logics "marathon-sim/logic"
	"os"
	"testing"

	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var broadcastTestConfig *model.Config

var broadcastTestEncounter12 *model.Encounter
var broadcastTestEncounter13 *model.Encounter

var broadcastTestLogger *logger.Logger

func SetupBroadcastLogicTest() {

	broadcastTestConfig = model.MakeDefaultConfig()

	filename := "configs/unit_tests/config_broadcast_test.json"
	filedata, err := os.ReadFile(filename)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err = json.Unmarshal(filedata, &broadcastTestConfig); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	broadcastTestLogger = logger.New()
	level, err := logger.ParseLevel(broadcastTestConfig.TopLevel.Log)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	broadcastTestLogger.SetLevel(level)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	broadcastTestLogger.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	broadcastTestLogger.Info("running with config:\n", broadcastTestConfig)

	// set up PRNG
	model.Seed(int64(broadcastTestConfig.TopLevel.Seed))
	broadcastTestLogger.Infof("random seed is %v", broadcastTestConfig.TopLevel.Seed)
	broadcastTestLogger.Info("starting")
	broadcastTestLogger.Infof("logging at log level %v; all times in UTC", broadcastTestConfig.TopLevel.Log)
	if path, err := os.Getwd(); err != nil {
		broadcastTestLogger.Fatalf("cannot get working directory: %v", err)
	} else {
		broadcastTestLogger.Debugf("running from %v", path)
	}

	// initializing the lenses
	lens.LensInit(broadcastTestLogger)

	// initialize the database
	model.Init(broadcastTestLogger, broadcastTestConfig)

	// initialize the message queue
	logics.DeliveredMessagesQueue = model.NewFastQueue()

	// set up channels 
	messageDBChan = make(chan *model.MessageDB, 1000)                  // buffer size of 1000 is arbitrary
	receivedmessageDBChan = make(chan *model.DeliveredMessageDB, 1000) // buffer size of 1000 is arbitrary
	logics.AssignChannels(messageDBChan, receivedmessageDBChan)


	// initialize the message counter
	logics.InitCounter(broadcastTestLogger)

	// initiailize the encounter manager - now consant of 20 seconds
	InitEncManager(float64(broadcastTestConfig.Simulation.TimeStep))

	logics.InitMemory(broadcastTestLogger, broadcastTestConfig)
	logics.LogicEnginesInit(broadcastTestLogger, broadcastTestConfig)

	// create a test encounter between two nodes

	broadcastTestEncounter12 = &model.Encounter{
		DatasetName:    "test dataset",
		Distance:       25.0,
		ExperimentName: "test experiment",
		Time:           1200000000,
		Node1:          1,
		Node2:          2,
		Duration:       10.0,
		X:              0.0,
		Y:              0.0,
		Z:              0.0,
	}

	broadcastTestEncounter13 = &model.Encounter{
		DatasetName:    "test dataset",
		Distance:       25.0,
		ExperimentName: "test experiment",
		Time:           1200000000,
		Node1:          1,
		Node2:          3,
		Duration:       10.0,
		X:              100.0,
		Y:              100.0,
		Z:              0.0,
	}

	// initialize node memory for our three test nodes

	logics.InitNodeMemory(broadcastTestConfig, broadcastTestEncounter12.Node1)
	logics.InitNodeMemory(broadcastTestConfig, broadcastTestEncounter12.Node2)
	logics.InitNodeMemory(broadcastTestConfig, broadcastTestEncounter13.Node2)
	// kick off helper function for messages channel
	logics.InitChan()
}

func TestBroadcastHandleEncounter(t *testing.T) {

	broadcastLogic := new(logics.BroadcastLogic)

	// test case 1: send one message between two nodes

	t.Run("SendOneMessageBetweenTwoNodes", func(t *testing.T) {

		SetupBroadcastLogicTest()

		message := new(logics.Message)
		// initialize node 1 with a message
		message = &logics.Message{
			MessageId:       "test message",
			Sender:          1,
			Type:            10,
			Destination:     2,
			DestinationNode: 2,
			Payload:         "test payload",
			MPop:            -1,
			LPop:            -1,
			CreationTime:    1200000000,
			TTLHops:         20,
			TTLTime:         1000000,
			LatHops:         20,
			Size:            1,
		}

		logics.UpdateMemoryNode(broadcastTestConfig, 1, message)

		bandwidth, _ := broadcastLogic.HandleEncounter(broadcastTestConfig, broadcastTestEncounter12)

		t.Logf("bandwidth: %v", bandwidth)

		node2Messages := logics.GetMessageQueue(2)

		messageCounter := 0

		node2Messages.Range(func(key, value interface{}) bool {
			messageCounter++
			t.Logf("Key: %v, Value: %v", key, value)
			return true
		})

		t.Logf("node 2 message count: %v", messageCounter)

		assert.Equal(t, 1, messageCounter)
		assert.Equal(t, float32(1.0), bandwidth)

	})

	// test case 2: send a bunch of messages between two nodes

	t.Run("SendManyMessageBetweenTwoNodes", func(t *testing.T) {

		SetupBroadcastLogicTest()

		// initialize node 1 with a 10 messages

		for i := 0; i < 10; i++ {

			message := new(logics.Message)

			message = &logics.Message{
				MessageId:       "test message " + fmt.Sprint(i),
				Sender:          1,
				Type:            10,
				Destination:     2,
				DestinationNode: 2,
				Payload:         "test payload",
				MPop:            -1,
				LPop:            -1,
				CreationTime:    1200000000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(broadcastTestConfig, 1, message)
		}

		bandwidth, _ := broadcastLogic.HandleEncounter(broadcastTestConfig, broadcastTestEncounter12)

		t.Logf("bandwidth: %v", bandwidth)

		node2Messages := logics.GetMessageQueue(2)

		messageCounter := 0

		node2Messages.Range(func(key, value interface{}) bool {
			messageCounter++
			t.Logf("Key: %v, Value: %v", key, value)
			return true
		})

		t.Logf("node 2 message count: %v", messageCounter)

		assert.Equal(t, 10, messageCounter)
		assert.Equal(t, float32(10.0), bandwidth)
	})

	// test case 3: send a bunch of messages between two nodes, excluding a third node

	t.Run("SendManyMessageBetweenTwoNodesExcludingThird", func(t *testing.T) {

		SetupBroadcastLogicTest()

		// initialize node 1 with a 10 messages

		for i := 0; i < 10; i++ {

			message := new(logics.Message)

			message = &logics.Message{
				MessageId:       "test message " + fmt.Sprint(i),
				Sender:          1,
				Type:            10,
				Destination:     2,
				DestinationNode: 2,
				Payload:         "test payload",
				MPop:            -1,
				LPop:            -1,
				CreationTime:    1200000000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(broadcastTestConfig, 1, message)
		}

		bandwidth, _ := broadcastLogic.HandleEncounter(broadcastTestConfig, broadcastTestEncounter12)

		t.Logf("bandwidth: %v", bandwidth)

		node2Messages := logics.GetMessageQueue(2)

		messageCounter2 := 0

		node2Messages.Range(func(key, value interface{}) bool {
			messageCounter2++
			t.Logf("Key: %v, Value: %v", key, value)
			return true
		})

		t.Logf("node 2 message count: %v", messageCounter2)

		node3Messages := logics.GetMessageQueue(3)

		messageCounter3 := 0

		node3Messages.Range(func(key, value interface{}) bool {
			messageCounter3++
			t.Logf("Key: %v, Value: %v", key, value)
			return true
		})

		t.Logf("node 3 message count: %v", messageCounter3)

		assert.Equal(t, 10, messageCounter2)
		assert.Equal(t, 0, messageCounter3)
		assert.Equal(t, float32(10.0), bandwidth)

	})

	// test case 4: two nodes send a bunch of messages to one node

	t.Run("TwoNodeSendingToOneNode", func(t *testing.T) {

		SetupBroadcastLogicTest()

		// initialize node 2 with 5 messages

		for i := 0; i < 5; i++ {

			message := new(logics.Message)

			message = &logics.Message{
				MessageId:       "test message from 2" + fmt.Sprint(i),
				Sender:          2,
				Type:            10,
				Destination:     1,
				DestinationNode: 1,
				Payload:         "test payload",
				MPop:            -1,
				LPop:            -1,
				CreationTime:    1200000000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(broadcastTestConfig, 2, message)
		}

		// initialize node 3 with 20 messages
		for i := 0; i < 20; i++ {

			message := new(logics.Message)

			message = &logics.Message{
				MessageId:       "test message from 3" + fmt.Sprint(i),
				Sender:          3,
				Type:            10,
				Destination:     1,
				DestinationNode: 1,
				Payload:         "test payload",
				MPop:            -1,
				LPop:            -1,
				CreationTime:    1200000000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(broadcastTestConfig, 3, message)
		}

		bandwidth12, _ := broadcastLogic.HandleEncounter(broadcastTestConfig, broadcastTestEncounter12)

		// now node 1 has 5 messages from node 2
		// during second encounter, these 5 messages are also passed to node 3

		bandwidth13, _ := broadcastLogic.HandleEncounter(broadcastTestConfig, broadcastTestEncounter13)

		t.Logf("bandwidth nodes 1, 2: %v", bandwidth12)
		t.Logf("bandwidth nodes 1, 3: %v", bandwidth13)

		node1Messages := logics.GetMessageQueue(1)

		messageCounter1 := 0

		node1Messages.Range(func(key, value interface{}) bool {
			messageCounter1++
			t.Logf("Key: %v, Value: %v", key, value)
			return true
		})

		t.Logf("node 1 message count: %v", messageCounter1)

		assert.Equal(t, 25, messageCounter1)
		assert.Equal(t, float32(5.0), bandwidth12)
		assert.Equal(t, float32(25.0), bandwidth13)
	})

	// test case 5: one node sending a bunch of messages to two nodes

	t.Run("OneNodeSendingToTwoNodes", func(t *testing.T) {

		SetupBroadcastLogicTest()

		// initialize node 1 with 20 messages

		for i := 0; i < 20; i++ {

			message := new(logics.Message)

			message = &logics.Message{
				MessageId:       "test message from 1" + fmt.Sprint(i),
				Sender:          1,
				Type:            10,
				Destination:     2,
				DestinationNode: 2,
				Payload:         "test payload",
				MPop:            -1,
				LPop:            -1,
				CreationTime:    1200000000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(broadcastTestConfig, 1, message)
		}

		bandwidth12, _ := broadcastLogic.HandleEncounter(broadcastTestConfig, broadcastTestEncounter12)
		bandwidth13, _ := broadcastLogic.HandleEncounter(broadcastTestConfig, broadcastTestEncounter13)

		t.Logf("bandwidth nodes 1, 2: %v", bandwidth12)
		t.Logf("bandwidth nodes 1, 3: %v", bandwidth13)

		node2Messages := logics.GetMessageQueue(2)

		messageCounter2 := 0

		node2Messages.Range(func(key, value interface{}) bool {
			messageCounter2++
			t.Logf("Key: %v, Value: %v", key, value)
			return true
		})

		node3Messages := logics.GetMessageQueue(3)

		messageCounter3 := 0

		node3Messages.Range(func(key, value interface{}) bool {
			messageCounter3++
			t.Logf("Key: %v, Value: %v", key, value)
			return true
		})

		t.Logf("node 1 message count: %v", messageCounter2)
		t.Logf("node 1 message count: %v", messageCounter3)

		assert.Equal(t, 20, messageCounter2)
		assert.Equal(t, 20, messageCounter3)
		assert.Equal(t, float32(20.0), bandwidth12)
		assert.Equal(t, float32(20.0), bandwidth13)
	})
}
