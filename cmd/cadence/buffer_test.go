/*

unit tests to check buffer eviction behavior

for simplicity, we will be testing buffer eviction using buffer logic as the underlying logic

*/

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

var bufferTestConfig *model.Config

var bufferTestEncounter12 *model.Encounter
var bufferTestEncounter13 *model.Encounter

var bufferTestLogger *logger.Logger

func SetupBufferTest() {

	bufferTestConfig = model.MakeDefaultConfig()

	// this special config file has buffer size set to 5
	filename := "configs/unit_tests/config_buffer_test.json"
	filedata, err := os.ReadFile(filename)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err = json.Unmarshal(filedata, &bufferTestConfig); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	bufferTestLogger = logger.New()
	level, err := logger.ParseLevel(bufferTestConfig.TopLevel.Log)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	bufferTestLogger.SetLevel(level)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	bufferTestLogger.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	bufferTestLogger.Info("running with config:\n", bufferTestConfig)

	// set up PRNG
	model.Seed(int64(bufferTestConfig.TopLevel.Seed))
	bufferTestLogger.Infof("random seed is %v", bufferTestConfig.TopLevel.Seed)
	bufferTestLogger.Info("starting")
	bufferTestLogger.Infof("logging at log level %v; all times in UTC", bufferTestConfig.TopLevel.Log)
	if path, err := os.Getwd(); err != nil {
		bufferTestLogger.Fatalf("cannot get working directory: %v", err)
	} else {
		bufferTestLogger.Debugf("running from %v", path)
	}

	// initializing the lenses
	lens.LensInit(bufferTestLogger)

	// initialize the database
	model.Init(bufferTestLogger, bufferTestConfig)

	// initialize the message queue
	logics.DeliveredMessagesQueue = model.NewFastQueue()

	// set up channels
	messageDBChan = make(chan *model.MessageDB, 1000)                  // buffer size of 1000 is arbitrary
	receivedmessageDBChan = make(chan *model.DeliveredMessageDB, 1000) // buffer size of 1000 is arbitrary
	logics.AssignChannels(messageDBChan, receivedmessageDBChan)

	// initialize the message counter
	logics.InitCounter(bufferTestLogger)

	// initiailize the encounter manager - now consant of 20 seconds
	InitEncManager(float64(bufferTestConfig.Simulation.TimeStep))

	logics.InitMemory(bufferTestLogger, bufferTestConfig)
	logics.LogicEnginesInit(bufferTestLogger, bufferTestConfig)

	// create a test encounter between two nodes

	bufferTestEncounter12 = &model.Encounter{
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

	bufferTestEncounter13 = &model.Encounter{
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

	logics.InitNodeMemory(bufferTestConfig, bufferTestEncounter12.Node1)
	logics.InitNodeMemory(bufferTestConfig, bufferTestEncounter12.Node2)
	logics.InitNodeMemory(bufferTestConfig, bufferTestEncounter13.Node2)
	// kick off helper function for messages channel
	logics.InitChan()
}

func TestBufferHandleEncounter(t *testing.T) {

	broadcastLogic := new(logics.BroadcastLogic)

	// check if the buffer size is set properly
	t.Run("ConfirmBufferSize", func(t *testing.T) {

		SetupBufferTest()

		node1Buffer := logics.GetMessageQueue(1)

		assert.NotNil(t, node1Buffer, "node 1 buffer should not be nil")
		node1BufferSize := logics.GetSimpleBuffer(1).BufferSize
		assert.Equal(t, float32(5), node1BufferSize, "node 1 buffer size should be 5")

		// check that the buffer is initially empty 
		assert.Equal(t, 0, CountSyncMapItems(node1Buffer), "node 1 buffer should be empty")
	})

	// partially load the buffer
	t.Run("SemiLoadBuffer", func(t *testing.T) {

		SetupBufferTest()

		node1Buffer := logics.GetMessageQueue(1)

		// try to add 3 messages to the buffer, and then check size 

		for i := 0; i < 3; i++ {

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
			logics.UpdateMemoryNode(bufferTestConfig, 1, message)
		}

		assert.Equal(t, 3, CountSyncMapItems(node1Buffer), "node 1 buffer should have 5 messages")
	})

	// fully load the buffer
	t.Run("FullLoadBuffer", func(t *testing.T) {

		SetupBufferTest()

		node1Buffer := logics.GetMessageQueue(1)

		// try to add 5 messages to the buffer, and then check size 

		for i := 0; i < 5; i++ {

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
			logics.UpdateMemoryNode(bufferTestConfig, 1, message)
		}

		assert.Equal(t, 5, CountSyncMapItems(node1Buffer), "node 1 buffer should have 5 messages")
	})

	// overload the buffer and check that the size is still 5
	t.Run("OverLoadBuffer", func(t *testing.T) {

		SetupBufferTest()

		node1Buffer := logics.GetMessageQueue(1)

		drops := 0.0

		// try to add 3 messages to the buffer, and then check size 

		for i := 0; i < 6; i++ {

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
			didDrop := logics.UpdateMemoryNode(bufferTestConfig, 1, message)

			if didDrop {
				drops++
			}
		}

		assert.Equal(t, 5, CountSyncMapItems(node1Buffer), "node 1 buffer should have 5 messages")
		assert.Equal(t, 1.0, drops, "1 message should have been dropped")
	})

	// overload even more 
	t.Run("OverLoadMoreBuffer", func(t *testing.T) {

		SetupBufferTest()

		node1Buffer := logics.GetMessageQueue(1)
		drops := 0.0

		// try to add 3 messages to the buffer, and then check size 

		for i := 0; i < 25; i++ {

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
			didDrop := logics.UpdateMemoryNode(bufferTestConfig, 1, message)

			if didDrop {
				drops++
			}		
		}		

		assert.Equal(t, 5, CountSyncMapItems(node1Buffer), "node 1 buffer should have 5 messages")
		assert.Equal(t, 20.0, drops, "20 messages should have been dropped")
	})
	// test case 1: send one message between two nodes

	t.Run("SendOneMessageBetweenTwoNodes", func(t *testing.T) {

		SetupBufferTest()

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

		logics.UpdateMemoryNode(bufferTestConfig, 1, message)

		bandwidth, _ := broadcastLogic.HandleEncounter(bufferTestConfig, bufferTestEncounter12)

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

		SetupBufferTest()

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
			logics.UpdateMemoryNode(bufferTestConfig, 1, message)
		}

		bandwidth, drops := broadcastLogic.HandleEncounter(bufferTestConfig, bufferTestEncounter12)

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
		assert.Equal(t, float32(0.0), drops, "no messages should have been dropped")
	})

	// test case 3: send a bunch of messages between two nodes, excluding a third node

	t.Run("SendManyMessageBetweenTwoNodesExcludingThird", func(t *testing.T) {

		SetupBufferTest()

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
			logics.UpdateMemoryNode(bufferTestConfig, 1, message)
		}

		bandwidth, drops := broadcastLogic.HandleEncounter(bufferTestConfig, bufferTestEncounter12)

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
		assert.Equal(t, float32(0.0), drops, "no messages should have been dropped")

	})

	// test case 4: two nodes send a bunch of messages to one node

	t.Run("TwoNodeSendingToOneNode", func(t *testing.T) {

		SetupBufferTest()

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
			logics.UpdateMemoryNode(bufferTestConfig, 2, message)
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
			logics.UpdateMemoryNode(bufferTestConfig, 3, message)
		}

		bandwidth12, _ := broadcastLogic.HandleEncounter(bufferTestConfig, bufferTestEncounter12)

		// now node 1 has 5 messages from node 2
		// during second encounter, these 5 messages are also passed to node 3

		bandwidth13, _ := broadcastLogic.HandleEncounter(bufferTestConfig, bufferTestEncounter13)

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

		SetupBufferTest()

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
			logics.UpdateMemoryNode(bufferTestConfig, 1, message)
		}

		bandwidth12, _ := broadcastLogic.HandleEncounter(bufferTestConfig, bufferTestEncounter12)
		bandwidth13, _ := broadcastLogic.HandleEncounter(bufferTestConfig, bufferTestEncounter13)

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
