package main

import (
	"database/sql"
	"encoding/json"
	"io"
	model "marathon-sim/datamodel"
	logics "marathon-sim/logic"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"
	"fmt"
	"os"
)

//This function aims to create the directory for message file if it is not existed.
func createMessageDirectory (msgFilePath string)(bool) {
	dirPath := filepath.Dir(msgFilePath)
	_, error := os.Stat(dirPath)
	if os.IsNotExist(error){
		fmt.Println("Directory is not existed") 
		if err := os.MkdirAll(dirPath, os.ModePerm); err == nil { 
			fmt.Println("Directory is succesfully created") 
			return true
		} else {
			log.Fatalf("Error creating directory: %v", err)
			return false
		}
	} else if error != nil {
			// Some other error occurred
		log.Fatalf("Error checking directory: %v", error)
		return false
	} else {
		fmt.Println("Directory is existed") 
		return true
	}
}

//This function parse the template json file to message for the later message generation use.
func parseMessageTemplate(filename string)(messageTemp []logics.Message, err error){
	var messages []logics.Message
	var message logics.Message
	dir, err := os.Getwd()
    if err != nil {
        return nil, fmt.Errorf("could not get current directory: %v", err)
    }

    // Build the full file path
    filepath := fmt.Sprintf("%s/%s", dir, filename)

	jsonFile, err := os.Open(filepath)
    if err != nil {
        log.Infof("the file cannot be opend due to %v", err)
        return nil, err
    }
    defer jsonFile.Close()

    // Read the file content
    byteValue, err := io.ReadAll(jsonFile)
    if err != nil {
		log.Infof("the file cannot be read due to %v", err)
        return nil, err
    }

    // Parse the JSON data
    err = json.Unmarshal(byteValue, &messages)
	
    if err != nil {
		error := json.Unmarshal(byteValue, &message)
		if error != nil {
			log.Infof("the file cannot be turned into list due to %v", error)
			return nil, error
		}
		messages = append(messages, message)
	}
	return messages, nil
}

//This function makes a sql query to get the information of nodes and its first time appear in the specific time span.
func getNodesData(ktop_nodes int, datasetName string, startTime, endTime float64, distance float64) (nodesTime map[model.NodeId]float64, er error){
	var err error
	var rows *sql.Rows
	nodesData := make(map[model.NodeId]float64)

	if ktop_nodes > 1 {
		// only get the top k nodes
		rows, err = model.DB.Table("events").
			Select("node, min(time), COUNT(*) as node_count").
			Where("dataset_name=? and time>=? and time<=?", datasetName, startTime, endTime).
			Group("node").
			Order("node_count DESC").
			Limit(ktop_nodes).
			Rows()
		if err != nil {
			log.Infof("error in fetching the ktop nodes %v", err)
			return nodesData, err
		}

		for rows.Next() {
			var node model.NodeId
			var count int
			var time float64

			if err := rows.Scan(&node, &time, &count); err != nil {
				log.Infof("invalid node: %v", err)
				continue
			}
			nodesData[node] = time
		}
	} else {
		rows, err = model.DB.Table("encounters").
			Select("node1, node2, time").
			Where("dataset_name=? and time>=? and time<=? and distance=?", datasetName, startTime, endTime, distance).
			Order("time").
			Rows()
		if err != nil {
			log.Infof("error in fetching the ktop nodes %v", err)
			return nodesData, err
		}
	
		for rows.Next() {
			var node1 model.NodeId
			var node2 model.NodeId
			var time float64
	
			if err := rows.Scan(&node1, &node2, &time); err != nil {
				log.Infof("invalid node: %v", err)
				continue
			}
			if prevTime, existed := nodesData[node1]; existed {
				nodesData[node1] = min(time, prevTime)
			} else {
				nodesData[node1] = time
			}
			if prevTime, existed := nodesData[node2]; existed {
				nodesData[node2] = min(time, prevTime)
			} else {
				nodesData[node2] = time
			}
				
		}
	}
	return nodesData, nil
}

//This function pairs up the nodes in the list and generate the one on one message between pairs based on the given template.
func generateDirectMessage(fileName string, nodesData map[model.NodeId]float64, minMessageSize, maxMessageSize float32, endTime float64)(map[model.NodeId][]logics.Message) {
	messages, err := parseMessageTemplate(fileName)
	var messageDataset = make(map[model.NodeId][]logics.Message)
	msgID := 0
	type communication struct{
		Sender model.NodeId
		Receiver model.NodeId
	}
	var receiverPool []model.NodeId
	//Copy the nodeData to get the potential receivers' list.
	for node := range nodesData {
		receiverPool =  append(receiverPool,node)
	}
	
	if err != nil{
		log.Infof("File cannot be parsed: %v", err)
	} 

	//pairing
	var communicationPair communication
	/* handle odd amount of nodes
	if len(receiverPool)!= 0{
		communicationPair.Sender = model.NodeIdInt(receiver)
	}*/
	for _, messageTemp := range(messages) {
		messageTemp.LatHops = 0
		for senderNode, senderTime := range(nodesData){
			// Pairing
			var repeatCheck bool = true
			var receiverNode model.NodeId
			for repeatCheck{
				receiverNode = generateRandomNode(receiverPool)
				if (receiverNode != senderNode){
					repeatCheck = false
				}
			}
			communicationPair.Sender = senderNode
			communicationPair.Receiver = receiverNode
			receiverPool = removeUsedReceiver(receiverPool, int(receiverNode))
			// Assign value			
			messageTemp.Sender = senderNode
			messageTemp.CreationTime = senderTime
			source := rand.NewSource(time.Now().UnixNano())
			//TO-DO: rewrite this to an individual function since group message also this func
			rng := rand.New(source)
			randomValue := rng.Float32()
			messageTemp.Size = minMessageSize + randomValue * (maxMessageSize - minMessageSize)
			receiver := communicationPair.Receiver
			messageTemp.MessageId = strconv.Itoa(msgID)
			if messageTemp.ShardsAvailable == 1{
				messageTemp.MessageId += "_"+ strconv.Itoa(messageTemp.ShardID)
			}
			messageTemp.Destination = receiver
			messageTemp.DestinationNode = model.NodeId(receiver)
			messageDataset[senderNode] = append(messageDataset[senderNode], messageTemp)
			msgID += 1
		}
	}
	return messageDataset
}


// This function generates a bunch of direct messages between nodes in the dataset 
// by default, each node will select ten random nodes to send direct messages to 
// message selection should probably be parameterized eventually
func generateBulkDirectMessages(fileName string, nodesData map[model.NodeId]float64, minMessageSize, maxMessageSize float32, endTime float64)(map[model.NodeId][]logics.Message) {
	messages, err := parseMessageTemplate(fileName)
	messageTemp := messages[0] // we only use the first message template
	var messageDataset = make(map[model.NodeId][]logics.Message)
	msgID := 0

	// number of messages per node
	// temporarily hardcoded to 10
	numMessagesPerNode := 10

	var receiverPool []model.NodeId
	//Copy the nodeData to get the potential receivers' list.
	for node := range nodesData {
		receiverPool =  append(receiverPool,node)
	}
	
	if err != nil{
		log.Infof("File cannot be parsed: %v", err)
	} 
	messageTemp.LatHops = 0
	for senderNode, senderTime := range(nodesData){

		// for each node we select as sender, we will select 10 random receivers 
		// Pairing

		receivers := generateRandomNodes(senderNode, receiverPool, numMessagesPerNode)
		if len(receivers) == 0 {
			log.Infof("No receivers available for node %v", senderNode)
			continue
		}

		fmt.Printf("Node %v is sending messages to %d receivers\n", senderNode, len(receivers))

		for _, receiverNode := range receivers {
			// create the message 

			messageTemp.Sender = senderNode
			messageTemp.CreationTime = senderTime

			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)
			randomValue := rng.Float32()
			messageTemp.Size = minMessageSize + randomValue * (maxMessageSize - minMessageSize)

			messageTemp.Destination = receiverNode
			messageTemp.DestinationNode = receiverNode

			messageTemp.MessageId = strconv.Itoa(msgID)
			if messageTemp.ShardsAvailable == 1{
				messageTemp.MessageId += "_"+ strconv.Itoa(messageTemp.ShardID)
			}
			
			messageDataset[senderNode] = append(messageDataset[senderNode], messageTemp)
			msgID += 1
		}
	}

	return messageDataset
}


// This function generates a bunch of direct messages between nodes in the dataset 
// by default, each node will select ten random nodes to send direct messages to 
// message selection should probably be parameterized eventually
func generateGradualBulkDirectMessages(fileName string, nodesData map[model.NodeId]float64, minMessageSize, maxMessageSize float32, endTime float64)(map[model.NodeId][]logics.Message) {
	messages, err := parseMessageTemplate(fileName)
	messageTemp := messages[0] // we only use the first message template
	var messageDataset = make(map[model.NodeId][]logics.Message)
	msgID := 0

	// number of messages per node
	// temporarily hardcoded to 10
	numMessagesPerNode := 10

	var receiverPool []model.NodeId
	//Copy the nodeData to get the potential receivers' list.
	for node := range nodesData {
		receiverPool =  append(receiverPool,node)
	}
	
	if err != nil{
		log.Infof("File cannot be parsed: %v", err)
	} 
	messageTemp.LatHops = 0
	for senderNode, senderTime := range(nodesData){

		// for each node we select as sender, we will select 10 random receivers 
		// Pairing

		receivers := generateRandomNodes(senderNode, receiverPool, numMessagesPerNode)
		if len(receivers) == 0 {
			log.Infof("No receivers available for node %v", senderNode)
			continue
		}

		fmt.Printf("Node %v is sending messages to %d receivers\n", senderNode, len(receivers))

		for _, receiverNode := range receivers {
			// create the message 

			// we generate the message gradually throughout the time span
			// by taking a random number from 0 to 1, multiplying it by the 
			// the span, and adding it as an offset to the sender's time

			messageTemp.CreationTime = senderTime + rand.Float64() * (endTime - senderTime)

			messageTemp.Sender = senderNode

			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)
			randomValue := rng.Float32()
			messageTemp.Size = minMessageSize + randomValue * (maxMessageSize - minMessageSize)

			messageTemp.Destination = receiverNode
			messageTemp.DestinationNode = receiverNode

			messageTemp.MessageId = strconv.Itoa(msgID)
			if messageTemp.ShardsAvailable == 1{
				messageTemp.MessageId += "_"+ strconv.Itoa(messageTemp.ShardID)
			}
			
			messageDataset[senderNode] = append(messageDataset[senderNode], messageTemp)
			msgID += 1
		}
	}

	return messageDataset
}


// This function generates a bunch of direct messages between nodes in the dataset 
// by default, each node will select ten random nodes to send direct messages to 
// message selection should probably be parameterized eventually
func generateHalfGradualBulkDirectMessages(fileName string, nodesData map[model.NodeId]float64, minMessageSize, maxMessageSize float32, endTime float64)(map[model.NodeId][]logics.Message) {
	messages, err := parseMessageTemplate(fileName)
	messageTemp := messages[0] // we only use the first message template
	var messageDataset = make(map[model.NodeId][]logics.Message)
	msgID := 0

	// number of messages per node
	// temporarily hardcoded to 10
	numMessagesPerNode := 10

	var receiverPool []model.NodeId
	//Copy the nodeData to get the potential receivers' list.
	for node := range nodesData {
		receiverPool =  append(receiverPool,node)
	}
	
	if err != nil{
		log.Infof("File cannot be parsed: %v", err)
	} 
	messageTemp.LatHops = 0
	for senderNode, senderTime := range(nodesData){

		// for each node we select as sender, we will select 10 random receivers 
		// Pairing

		receivers := generateRandomNodes(senderNode, receiverPool, numMessagesPerNode)
		if len(receivers) == 0 {
			log.Infof("No receivers available for node %v", senderNode)
			continue
		}

		fmt.Printf("Node %v is sending messages to %d receivers\n", senderNode, len(receivers))

		for _, receiverNode := range receivers {
			// create the message 
			// we generate the message gradually throughout the time span
			// by taking a random number from 0 to 1, multiplying it by the 
			// the span, and adding it as an offset to the sender's time

			// in this version, we only generate messages for the first half of the time span

			messageTemp.CreationTime = senderTime + rand.Float64() * (endTime - senderTime) * 0.5

			messageTemp.Sender = senderNode

			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)
			randomValue := rng.Float32()
			messageTemp.Size = minMessageSize + randomValue * (maxMessageSize - minMessageSize)

			messageTemp.Destination = receiverNode
			messageTemp.DestinationNode = receiverNode

			messageTemp.MessageId = strconv.Itoa(msgID)
			if messageTemp.ShardsAvailable == 1{
				messageTemp.MessageId += "_"+ strconv.Itoa(messageTemp.ShardID)
			}
			
			messageDataset[senderNode] = append(messageDataset[senderNode], messageTemp)
			msgID += 1
		}
	}

	return messageDataset
}

//This is a helper function to help 1 on 1 message generation to randomly pair up sender and receiver.
func generateRandomNode(receiverPool []model.NodeId)(model.NodeId){
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	randomValue := rng.Intn(len(receiverPool))
	return receiverPool[randomValue]
}

// this function takes in a slice of receiver nodes and selects randomly n unique nodes from it,
// ensuring that the senderNode is not included in the receivers list
func generateRandomNodes(senderNode model.NodeId, receiverPool []model.NodeId, n int) []model.NodeId {
	// Filter out the senderNode from the receiverPool
	filteredPool := make([]model.NodeId, 0, len(receiverPool))
	for _, node := range receiverPool {
		if node != senderNode {
			filteredPool = append(filteredPool, node)
		}
	}

	if n > len(filteredPool) {
		n = len(filteredPool)
	}

	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)


	// selected nodes is a map. It is used to ensure uniqueness 
	// we select a random node from the pool and add it to the map.
	// if the node we selected is already in the map, it "overwrites" 
	// the previous value and the map DOES NOT GROW IN SIZE. 
	// the termination condition for this loop is the size of the map. 
	selectedNodes := make(map[model.NodeId]bool)
	for len(selectedNodes) < n {
		randomIndex := rng.Intn(len(filteredPool))
		selectedNode := filteredPool[randomIndex]
		selectedNodes[selectedNode] = true
	}

	result := make([]model.NodeId, 0, len(selectedNodes))
	for node := range selectedNodes {
		result = append(result, node)
	}
	return result
}

//This is a helper function to remove the receiver nodes from the pool that already been paired up.
func removeUsedReceiver(receiverPool []model.NodeId, valueToRemove int)([]model.NodeId){
	var result []model.NodeId

    // Iterate over the original slice
    for _, value := range receiverPool {
        // Append to result if value is not the one to remove
        if int(value) != valueToRemove {
            result = append(result, value)
        }
    }

    return result
}

//This is message generation function 
func generateGroupMessage(fileName string, nodesData map[model.NodeId]float64, minMessageSize, maxMessageSize float32, endTime float64)(map[model.NodeId][]logics.Message) {
	messages, err := parseMessageTemplate(fileName)
	var messageDataset = make(map[model.NodeId][]logics.Message)
	msgID := 0
	if err != nil{
		log.Infof("File cannot be parsed: %v", err)
	}
	for _, messageTemp := range(messages){
		messageTemp.LatHops = 0
		for senderNode, senderTime := range nodesData {
			messageTemp.Sender = senderNode
			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)
			randomValue := rng.Float32()
			messageTemp.Size = minMessageSize + randomValue * (maxMessageSize - minMessageSize)
			timer := 0.0
			for receiverNode := range(nodesData){
				if senderNode == receiverNode{
					continue
				}
				messageTemp.CreationTime = senderTime + timer
				timer += 10.0
				messageTemp.MessageId = strconv.Itoa(msgID)
				if messageTemp.ShardsAvailable == 1{
					messageTemp.MessageId += "_"+ strconv.Itoa(messageTemp.ShardID)
				}
				messageTemp.Destination = receiverNode
				messageTemp.DestinationNode = receiverNode
				messageDataset[senderNode] = append(messageDataset[senderNode], messageTemp)
				msgID += 1

			}
		}
	}
	return messageDataset
}

//This is function preparing for the message generation before the actaul delievery
func messageGenerate(datasetName string, messagesTemplatePath string, 
	min_message_size, max_message_size float32, startTime, endTime float64, 
	messagesFile string, messageGeneratorOption string, distance float64, ktop int) {

	// Make a SQL query to get all nodes we need for message
	nodesData, error := getNodesData(ktop, datasetName, startTime, endTime, distance)
	if error != nil {
		log.Infof("nodes cannot be accessed because of  %v", error)
	}


	// Generate messages
	var messages map[model.NodeId][]logics.Message
	switch messageGeneratorOption {
	case "0":
		messages = generateDirectMessage(messagesTemplatePath, nodesData, min_message_size, max_message_size, endTime)
	case "1":
		messages = generateGroupMessage(messagesTemplatePath, nodesData, min_message_size, max_message_size, endTime)
	case "2":
		messages = generateBulkDirectMessages(messagesTemplatePath, nodesData, min_message_size, max_message_size, endTime)
	case "3":
		messages = generateGradualBulkDirectMessages(messagesTemplatePath, nodesData, min_message_size, max_message_size, endTime)
	case "4":
		messages = generateHalfGradualBulkDirectMessages(messagesTemplatePath, nodesData, min_message_size, max_message_size, endTime)
	default:
		log.Infof("Invalid message generation option: %s", messageGeneratorOption)
		return
	}

	// Write messages to the output file
	writeFile(messagesFile, messages)

}

// This is a function to write message back to json file and save it to the assigned directory
func writeFile(messagesFile string, messages map[model.NodeId][]logics.Message) (bool){
	checkDirectory := createMessageDirectory(messagesFile)
	if checkDirectory{
		file, err := os.Create(messagesFile)
		if err != nil {
			fmt.Println("Error creating output file:", err)
			return false
		}
		defer file.Close()

		if err := json.NewEncoder(file).Encode(messages); err != nil {
			fmt.Println("Error writing messages to file:", err)
			return false
		}

		fmt.Println("Messages dumped to", messagesFile)
		return true
	} else {
		return false
	}
}

//This is a helper functions to find minimum value
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

