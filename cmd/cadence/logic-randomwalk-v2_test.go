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

var randomwalkV2TestConfig *model.Config

var randomwalkV2TestEncounter12 *model.Encounter
var randomwalkV2TestEncounter13 *model.Encounter

var randomwalkV2TestLogger *logger.Logger

func SetupRandomwalkV2LogicTest() {

	randomwalkV2TestConfig = model.MakeDefaultConfig()

	filename := "configs/unit_tests/config_randomwalkV2_test.json"
	filedata, err := os.ReadFile(filename)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err = json.Unmarshal(filedata, &randomwalkV2TestConfig); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	randomwalkV2TestLogger = logger.New()
	level, err := logger.ParseLevel(randomwalkV2TestConfig.TopLevel.Log)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	randomwalkV2TestLogger.SetLevel(level)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	randomwalkV2TestLogger.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	randomwalkV2TestLogger.Info("running with config:\n", randomwalkV2TestConfig)

	// set up PRNG
	model.Seed(int64(randomwalkV2TestConfig.TopLevel.Seed))
	randomwalkV2TestLogger.Infof("random seed is %v", randomwalkV2TestConfig.TopLevel.Seed)
	randomwalkV2TestLogger.Info("starting")
	randomwalkV2TestLogger.Infof("logging at log level %v; all times in UTC", randomwalkV2TestConfig.TopLevel.Log)
	if path, err := os.Getwd(); err != nil {
		randomwalkV2TestLogger.Fatalf("cannot get working directory: %v", err)
	} else {
		randomwalkV2TestLogger.Debugf("running from %v", path)
	}

	// initializing the lenses
	lens.LensInit(randomwalkV2TestLogger)

	// initialize the database
	model.Init(randomwalkV2TestLogger, randomwalkV2TestConfig)

	// initialize the message queue
	logics.DeliveredMessagesQueue = model.NewFastQueue()

	// set up channels 
	messageDBChan = make(chan *model.MessageDB, 1000)                  // buffer size of 1000 is arbitrary
	receivedmessageDBChan = make(chan *model.DeliveredMessageDB, 1000) // buffer size of 1000 is arbitrary
	logics.AssignChannels(messageDBChan, receivedmessageDBChan)

	// initialize the message counter
	logics.InitCounter(randomwalkV2TestLogger)

	// initiailize the encounter manager - now consant of 20 seconds
	InitEncManager(float64(randomwalkV2TestConfig.Simulation.TimeStep))

	logics.InitMemory(randomwalkV2TestLogger, randomwalkV2TestConfig)
	logics.LogicEnginesInit(randomwalkV2TestLogger, randomwalkV2TestConfig)

	// create a test encounter between two nodes

	randomwalkV2TestEncounter12 = &model.Encounter{
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

	randomwalkV2TestEncounter13 = &model.Encounter{
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

	logics.InitNodeMemory(randomwalkV2TestConfig, randomwalkV2TestEncounter12.Node1)
	logics.InitNodeMemory(randomwalkV2TestConfig, randomwalkV2TestEncounter12.Node2)
	logics.InitNodeMemory(randomwalkV2TestConfig, randomwalkV2TestEncounter13.Node2)
	// kick off helper function for messages channel
	logics.InitChan()

}

// test if config parameters are being properly set
// using json file
func TestRandomwalkV2InitLogic(t *testing.T) {

	// setup the config and the block

	randomwalkV2TestConfig = model.MakeDefaultConfig()

	filename := "configs/unit_tests/config_randomwalkV2_test.json"
	filedata, err := os.ReadFile(filename)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err = json.Unmarshal(filedata, &randomwalkV2TestConfig); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	logicFile := randomwalkV2TestConfig.Simulation.LogicFile

	// setup the logger

	randomwalkV2TestLogger = logger.New()
	level, err := logger.ParseLevel(randomwalkV2TestConfig.TopLevel.Log)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	randomwalkV2TestLogger.SetLevel(level)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	randomwalkV2TestLogger.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	// create the block
	block := logics.ConfigLogic(logicFile, randomwalkV2TestLogger)

	// check that block is not null

	assert.NotNil(t, block)

	// check that the block has correct information
	assert.Equal(t, "randomwalk-v2", block.LogicName)
	assert.Equal(t, float32(1000), block.ConstantTransferProbability)
	assert.Equal(t, float32(1.0), block.RandomwalkDeleteProbability)
}

func TestRandomwalkV2HandleEncounter(t *testing.T) {

	randomwalkV2Logic := new(logics.RandomWalkLogicV2)

	// test case 1 - 5 : send a bunch of messages between two nodes
	// with varying message creation times

	// test case 1: difference between message creation time and encounter
	// time is greater than time constant, so all messages should be
	// transferred
	t.Run("SendMessagesCase1", func(t *testing.T) {

		totalBandwidth := float32(0.0)

		// run 10 separate experiments

		for epoch := 0; epoch < 10; epoch++ {

			t.Logf("epoch: %v", epoch)
			SetupRandomwalkV2LogicTest()

			randomwalkV2Logic.SetConstant(1000)

			t.Logf("time constant: %v", randomwalkV2Logic.GetConstant())
			currentBandwidth := float32(0.0)

			for i := 0; i < 100; i++ {

				message := new(logics.Message)

				message = &logics.Message{
					MessageId:       "test message " + fmt.Sprint(i) + ", epoch: " + fmt.Sprint(epoch),
					Sender:          1,
					Type:            10,
					Destination:     2,
					DestinationNode: 2,
					Payload:         "test payload",
					MPop:            -1,
					LPop:            -1,
					CreationTime:    1199998000,
					TTLHops:         20,
					TTLTime:         1000000,
					LatHops:         20,
					Size:            1,
				}
				logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
			}

			currentBandwidth, _ = randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
			t.Logf("bandwidth: %v", currentBandwidth)
			totalBandwidth += currentBandwidth
		}

		averageBandwidth := totalBandwidth / 10.0

		t.Logf("average bandwidth: %v", averageBandwidth)

		assert.True(t, averageBandwidth == 100.0)
	})

	// test case 2: difference between message creation time and encounter
	// time is exactly the time constant, so all messages should be
	// transferred
	t.Run("SendMessagesCase2", func(t *testing.T) {

		totalBandwidth := float32(0.0)

		// run 10 separate experiments

		for epoch := 0; epoch < 10; epoch++ {

			t.Logf("epoch: %v", epoch)
			SetupRandomwalkV2LogicTest()

			randomwalkV2Logic.SetConstant(1000)

			t.Logf("time constant: %v", randomwalkV2Logic.GetConstant())
			currentBandwidth := float32(0.0)

			for i := 0; i < 100; i++ {

				message := new(logics.Message)

				message = &logics.Message{
					MessageId:       "test message " + fmt.Sprint(i) + ", epoch: " + fmt.Sprint(epoch),
					Sender:          1,
					Type:            10,
					Destination:     2,
					DestinationNode: 2,
					Payload:         "test payload",
					MPop:            -1,
					LPop:            -1,
					CreationTime:    1199999000,
					TTLHops:         20,
					TTLTime:         1000000,
					LatHops:         20,
					Size:            1,
				}
				logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
			}

			currentBandwidth, _ = randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
			t.Logf("bandwidth: %v", currentBandwidth)
			totalBandwidth += currentBandwidth
		}

		averageBandwidth := totalBandwidth / 10.0

		t.Logf("average bandwidth: %v", averageBandwidth)

		assert.True(t, averageBandwidth == 100.0)
	})

	// test case 3: time held is 8/10 of the time constant, so
	// approx 80% of messages should be transferred
	t.Run("SendMessagesCase3", func(t *testing.T) {

		totalBandwidth := float32(0.0)

		// run 10 separate experiments

		for epoch := 0; epoch < 10; epoch++ {

			t.Logf("epoch: %v", epoch)
			SetupRandomwalkV2LogicTest()

			randomwalkV2Logic.SetConstant(1000)

			t.Logf("time constant: %v", randomwalkV2Logic.GetConstant())
			currentBandwidth := float32(0.0)

			for i := 0; i < 100; i++ {

				message := new(logics.Message)

				message = &logics.Message{
					MessageId:       "test message " + fmt.Sprint(i) + ", epoch: " + fmt.Sprint(epoch),
					Sender:          1,
					Type:            10,
					Destination:     2,
					DestinationNode: 2,
					Payload:         "test payload",
					MPop:            -1,
					LPop:            -1,
					CreationTime:    1199999200,
					TTLHops:         20,
					TTLTime:         1000000,
					LatHops:         20,
					Size:            1,
				}
				logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
			}

			currentBandwidth, _ = randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
			t.Logf("bandwidth: %v", currentBandwidth)
			totalBandwidth += currentBandwidth
		}

		averageBandwidth := totalBandwidth / 10.0

		t.Logf("average bandwidth: %v", averageBandwidth)

		assert.True(t, averageBandwidth > 70.0 && averageBandwidth < 90.0)
	})

	// test case 4: time held is 6/10 of the time constant, so
	// approx 60% of messages should be transferred
	t.Run("SendMessagesCase4", func(t *testing.T) {

		totalBandwidth := float32(0.0)

		// run 10 separate experiments

		for epoch := 0; epoch < 10; epoch++ {

			t.Logf("epoch: %v", epoch)
			SetupRandomwalkV2LogicTest()

			randomwalkV2Logic.SetConstant(1000)

			t.Logf("time constant: %v", randomwalkV2Logic.GetConstant())
			currentBandwidth := float32(0.0)

			for i := 0; i < 100; i++ {

				message := new(logics.Message)

				message = &logics.Message{
					MessageId:       "test message " + fmt.Sprint(i) + ", epoch: " + fmt.Sprint(epoch),
					Sender:          1,
					Type:            10,
					Destination:     2,
					DestinationNode: 2,
					Payload:         "test payload",
					MPop:            -1,
					LPop:            -1,
					CreationTime:    1199999400,
					TTLHops:         20,
					TTLTime:         1000000,
					LatHops:         20,
					Size:            1,
				}
				logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
			}

			currentBandwidth, _ = randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
			t.Logf("bandwidth: %v", currentBandwidth)
			totalBandwidth += currentBandwidth
		}

		averageBandwidth := totalBandwidth / 10.0

		t.Logf("average bandwidth: %v", averageBandwidth)

		assert.True(t, averageBandwidth > 50.0 && averageBandwidth < 70.0)
	})

	// test case 5: time held is 4/10 of the time constant, so
	// approx 40% of messages should be transferred
	t.Run("SendMessagesCase5", func(t *testing.T) {

		totalBandwidth := float32(0.0)

		// run 10 separate experiments

		for epoch := 0; epoch < 10; epoch++ {

			t.Logf("epoch: %v", epoch)
			SetupRandomwalkV2LogicTest()

			randomwalkV2Logic.SetConstant(1000)

			t.Logf("time constant: %v", randomwalkV2Logic.GetConstant())
			currentBandwidth := float32(0.0)

			for i := 0; i < 100; i++ {

				message := new(logics.Message)

				message = &logics.Message{
					MessageId:       "test message " + fmt.Sprint(i) + ", epoch: " + fmt.Sprint(epoch),
					Sender:          1,
					Type:            10,
					Destination:     2,
					DestinationNode: 2,
					Payload:         "test payload",
					MPop:            -1,
					LPop:            -1,
					CreationTime:    1199999600,
					TTLHops:         20,
					TTLTime:         1000000,
					LatHops:         20,
					Size:            1,
				}
				logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
			}

			currentBandwidth, _ = randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
			t.Logf("bandwidth: %v", currentBandwidth)
			totalBandwidth += currentBandwidth
		}

		averageBandwidth := totalBandwidth / 10.0

		t.Logf("average bandwidth: %v", averageBandwidth)

		assert.True(t, averageBandwidth > 30.0 && averageBandwidth < 50.0)
	})

	// test case 6: time held is 2/10 of the time constant, so
	// approx 20% of messages should be transferred
	t.Run("SendMessagesCase6", func(t *testing.T) {

		totalBandwidth := float32(0.0)

		// run 10 separate experiments

		for epoch := 0; epoch < 10; epoch++ {

			t.Logf("epoch: %v", epoch)
			SetupRandomwalkV2LogicTest()

			randomwalkV2Logic.SetConstant(1000)

			t.Logf("time constant: %v", randomwalkV2Logic.GetConstant())
			currentBandwidth := float32(0.0)

			for i := 0; i < 100; i++ {

				message := new(logics.Message)

				message = &logics.Message{
					MessageId:       "test message " + fmt.Sprint(i) + ", epoch: " + fmt.Sprint(epoch),
					Sender:          1,
					Type:            10,
					Destination:     2,
					DestinationNode: 2,
					Payload:         "test payload",
					MPop:            -1,
					LPop:            -1,
					CreationTime:    1199999800,
					TTLHops:         20,
					TTLTime:         1000000,
					LatHops:         20,
					Size:            1,
				}
				logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
			}

			currentBandwidth, _ = randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
			t.Logf("bandwidth: %v", currentBandwidth)
			totalBandwidth += currentBandwidth
		}

		averageBandwidth := totalBandwidth / 10.0

		t.Logf("average bandwidth: %v", averageBandwidth)

		assert.True(t, averageBandwidth > 10.0 && averageBandwidth < 30.0)
	})

	// test case 7: encounter occurs at message creation time,
	// no messages should be passed
	t.Run("SendMessagesCase7", func(t *testing.T) {

		totalBandwidth := float32(0.0)

		// run 10 separate experiments

		for epoch := 0; epoch < 10; epoch++ {

			t.Logf("epoch: %v", epoch)
			SetupRandomwalkV2LogicTest()

			randomwalkV2Logic.SetConstant(1000)

			t.Logf("time constant: %v", randomwalkV2Logic.GetConstant())
			currentBandwidth := float32(0.0)

			for i := 0; i < 100; i++ {

				message := new(logics.Message)

				message = &logics.Message{
					MessageId:       "test message " + fmt.Sprint(i) + ", epoch: " + fmt.Sprint(epoch),
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
				logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
			}

			currentBandwidth, _ = randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
			t.Logf("bandwidth: %v", currentBandwidth)
			totalBandwidth += currentBandwidth
		}

		averageBandwidth := totalBandwidth / 10.0

		t.Logf("average bandwidth: %v", averageBandwidth)

		assert.True(t, averageBandwidth == 0.0)
	})

	// test case 8: send a bunch of messages between two nodes, excluding a third node

	t.Run("SendManyMessageBetweenTwoNodesExcludingThird", func(t *testing.T) {

		SetupRandomwalkV2LogicTest()
		// randomwalkV2Logic.SetTransferProbability(1.0)

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
				CreationTime:    1199999000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
		}

		bandwidth, _ := randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)

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

	// test case 9: two nodes send a bunch of messages to one node

	t.Run("TwoNodeSendingToOneNode", func(t *testing.T) {

		SetupRandomwalkV2LogicTest()
		// randomwalkV2Logic.SetTransferProbability(1.0)

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
				CreationTime:    1199999000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(randomwalkV2TestConfig, 2, message)
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
				CreationTime:    1199999000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(randomwalkV2TestConfig, 3, message)
		}

		bandwidth12, _ := randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)

		// now node 1 has 5 messages from node 2
		// during second encounter, these 5 messages are also passed to node 3

		bandwidth13, _ := randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter13)

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

	// test case 10: one node sending a bunch of messages to two nodes

	t.Run("OneNodeSendingToTwoNodes", func(t *testing.T) {

		SetupRandomwalkV2LogicTest()
		// randomwalkV2Logic.SetTransferProbability(1.0)

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
				CreationTime:    1199999000,
				TTLHops:         20,
				TTLTime:         1000000,
				LatHops:         20,
				Size:            1,
			}
			logics.UpdateMemoryNode(randomwalkV2TestConfig, 1, message)
		}

		bandwidth12, _ := randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter12)
		bandwidth13, _ := randomwalkV2Logic.HandleEncounter(randomwalkV2TestConfig, randomwalkV2TestEncounter13)

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
