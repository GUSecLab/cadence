package main

import (
	model "marathon-sim/datamodel"
	logics "marathon-sim/logic"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNodesData(t *testing.T) {
	setup()
	t.Run("getNodeandTimewithLimit10", func(t *testing.T) {
		//Tests used by Avery
		expectedNodes := map[model.NodeId]float64{516: 1211028573, 492: 1211022811,
			207: 1211018411, 192: 1211018404, 173: 1211018429, 448: 1211022225,
			194: 1211018453, 384: 1211022515, 510: 1211018456, 154: 1211018456}
		var ktopnode int = 10
		var start float64 = 1211018404.0
		var end float64 = 1213089934.0
		var distance float64 = 10.0
		nodeData, err := getNodesData(ktopnode, "cabspotting", start, end, distance)
		for node, time := range expectedNodes {
			assert.Equal(t, time, nodeData[node])
			assert.Nil(t, err)
		}

	})
	t.Run("getNodeandTimewithLimit1", func(t *testing.T) {
		expectedNodes := map[model.NodeId]float64{2: 323456.789,
			3: 323456.789}
		var ktopnode int = 0
		var start float64 = 223456.000
		var end float64 = 723456.0
		var distance float64 = 10.8
		nodeData, err := getNodesData(ktopnode, "cabspotting", start, end, distance)
		for node, time := range expectedNodes {
			assert.Equal(t, time, nodeData[node])
			assert.Nil(t, err)
		}
	})

}

func TestParseMessageTemplateGHCI(t *testing.T) {
	t.Run("parseOneMsgTemplate", func(t *testing.T) {
		var expectedMessages = []logics.Message{}
		filename := "messages_template.json"
		var destination float64 = -1
		expectedMessages = append(expectedMessages, logics.Message{
			MessageId:       "",
			Sender:          -1,
			Type:            10,
			CreationTime:    0,
			Destination:     destination,
			DestinationNode: 0, // Assuming DestinationNode should be same as Destination
			Payload:         "Hi ",
			ShardsAvailable: 0,
			ShardID:         0,
			TTLHops:         20,
			TTLTime:         1000000,
			LatHops:         0,
			Size:            1,
		})
		messages, error := parseMessageTemplate(filename)
		assert.Equal(t, expectedMessages, messages)
		assert.Nil(t, error)
	})

	t.Run("parseThreeMsgTemplate", func(t *testing.T) {
		var expectedMessages = []logics.Message{}
		filename := "messages_backup.json"
		var destination float64 = -1
		expectedMessages = append(expectedMessages, logics.Message{
			MessageId:       "",
			Sender:          -1,
			Type:            10,
			CreationTime:    0,
			Destination:     destination,
			DestinationNode: 0, // Assuming DestinationNode should be same as Destination
			Payload:         "Hi ",
			ShardsAvailable: 0,
			ShardID:         0,
			MShards:         false,
			TTLHops:         20,
			TTLTime:         1000,
			LatHops:         0,
			Size:            30,
		}, logics.Message{
			MessageId:       "",
			Sender:          -1,
			Type:            10,
			CreationTime:    0,
			Destination:     destination,
			DestinationNode: 0, // Assuming DestinationNode should be same as Destination
			Payload:         "Hello to ,",
			ShardsAvailable: 1,
			ShardID:         0,
			MShards:         true,
			TTLHops:         20,
			TTLTime:         1000,
			LatHops:         0,
			Size:            20,
		}, logics.Message{
			MessageId:       "",
			Sender:          -1,
			Type:            10,
			CreationTime:    0,
			Destination:     destination,
			DestinationNode: 0, // Assuming DestinationNode should be same as Destination
			Payload:         ". How are you?",
			ShardsAvailable: 1,
			ShardID:         1,
			MShards:         false,
			TTLHops:         20,
			TTLTime:         1000,
			LatHops:         0,
			Size:            20,
		})
		messages, error := parseMessageTemplate(filename)
		assert.Equal(t, expectedMessages, messages)
		assert.Nil(t, error)
	})
}

func TestGenerateDirectMessage(t *testing.T) {
	// This test can't be right since destination node is randomly generated
	filename := "messages_template.json"
	// var expectedResult = make(map[model.NodeId][]logics.Message)
	nodesData := map[model.NodeId]float64{516: 1211028573, 492: 1211022811,
		207: 1211018411, 192: 1211018404, 173: 1211018429, 448: 1211022225,
		194: 1211018453, 384: 1211022515, 510: 1211018456, 154: 1211018456}

	var minMessageSize float32 = 1.0
	var maxMessageSize float32 = 1.0
	result := generateDirectMessage(filename, nodesData, minMessageSize, maxMessageSize, 0)
	t.Logf("result: %v", result)
	// assert.Equal(t, expectedResult, result)
}


func TestGenerateMessage(t *testing.T) {
// This test goes wrong some time because iterating map with range is random-ordered
	filename := "messages_backup.json"
	var expectedResult = make(map[model.NodeId][]logics.Message)
	destination := model.NodeId(3)
	destination_2 := model.NodeId(2)

	expectedResult[2] = append(expectedResult[2], logics.Message{
		MessageId:       "0",
		Sender:          2,
		Type:            10,
		CreationTime:    323456.789,
		Destination:     destination,
		DestinationNode: 3, // Assuming DestinationNode should be same as Destination
		Payload:         "Hi ",
		ShardsAvailable: 0,
		ShardID:         0,
		MShards:         false,
		TTLHops:         20,
		TTLTime:         1000,
		LatHops:         0,
		Size:            1,
	}, logics.Message{
		MessageId:       "2_0",
		Sender:          2,
		Type:            10,
		CreationTime:    323456.789,
		Destination:     destination,
		DestinationNode: 3, // Assuming DestinationNode should be same as Destination
		Payload:         "Hello to ,",
		ShardsAvailable: 1,
		ShardID:         0,
		MShards:         true,
		TTLHops:         20,
		TTLTime:         1000,
		LatHops:         0,
		Size:            1,
	}, logics.Message{
		MessageId:       "4_1",
		Sender:          2,
		Type:            10,
		CreationTime:    323456.789,
		Destination:     destination,
		DestinationNode: 3, // Assuming DestinationNode should be same as Destination
		Payload:         ". How are you?",
		ShardsAvailable: 1,
		ShardID:         1,
		MShards:         false,
		TTLHops:         20,
		TTLTime:         1000,
		LatHops:         0,
		Size:            1,
	})
	expectedResult[3] = append(expectedResult[3], logics.Message{
		MessageId:       "1",
		Sender:          3,
		Type:            10,
		CreationTime:    323456.789,
		Destination:     destination_2,
		DestinationNode: 2, // Assuming DestinationNode should be same as Destination
		Payload:         "Hi ",
		ShardsAvailable: 0,
		ShardID:         0,
		MShards:         false,
		TTLHops:         20,
		TTLTime:         1000,
		LatHops:         0,
		Size:            1,
	}, logics.Message{
		MessageId:       "3_0",
		Sender:          3,
		Type:            10,
		CreationTime:    323456.789,
		Destination:     destination_2,
		DestinationNode: 2, // Assuming DestinationNode should be same as Destination
		Payload:         "Hello to ,",
		ShardsAvailable: 1,
		ShardID:         0,
		MShards:         true,
		TTLHops:         20,
		TTLTime:         1000,
		LatHops:         0,
		Size:            1,
	}, logics.Message{
		MessageId:       "5_1",
		Sender:          3,
		Type:            10,
		CreationTime:    323456.789,
		Destination:     destination_2,
		DestinationNode: 2, // Assuming DestinationNode should be same as Destination
		Payload:         ". How are you?",
		ShardsAvailable: 1,
		ShardID:         1,
		MShards:         false,
		TTLHops:         20,
		TTLTime:         1000,
		LatHops:         0,
		Size:            1,
	})
	nodesData := map[model.NodeId]float64{2: 323456.789, 3: 323456.789}
	var minMessageSize float32 = 1.0
	var maxMessageSize float32 = 1.0
	result := generateGroupMessage(filename, nodesData, minMessageSize, maxMessageSize, 0)
	assert.Equal(t, expectedResult, result)
}

func TestCreateMessageDirectoryGHCI(t *testing.T) {
	//need to rewrite this
	// Define the new directory path inside the temporary directory
	newDirPath := "../../marathon-files/messages_cab.json"
	dirPath := filepath.Dir(newDirPath)
	assert.Equal(t, "../../marathon-files", dirPath)
	// Call the function to create the new directory
	check := createMessageDirectory(newDirPath)

	assert.True(t, check)
}

func TestWriteFileGHCI(t *testing.T) {
	filename := "messages_backup.json"
	nodesData := map[model.NodeId]float64{2: 323456.789, 3: 323456.789}
	var minMessageSize float32 = 1.0
	var maxMessageSize float32 = 1.0
	messages := generateGroupMessage(filename, nodesData, minMessageSize, maxMessageSize, 0)
	result := writeFile("../../marathon-files/messages_cab.json", messages)
	assert.True(t, result)
}
