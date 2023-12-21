package logic

import (
	"encoding/json"
	"fmt"
	model "marathon-sim/datamodel"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	logger "github.com/sirupsen/logrus"
)

// type of message
type AddressType int64

const (
	AddressTypeUnknown   AddressType = 0
	AddressTypeUnicast   AddressType = 10
	AddressTypeMulticast AddressType = 20
	AddressTypeBroadcast AddressType = 30
	AddressTypeAnycast   AddressType = 40
)

func addressTypeToString(at AddressType) string {
	switch at {
	case AddressTypeUnknown:
		return "unknown"
	case AddressTypeUnicast:
		return "unicast"
	case AddressTypeMulticast:
		return "multicast"
	case AddressTypeBroadcast:
		return "broadcast"
	case AddressTypeAnycast:
		return "anycast"
	default:
		panic("unknown address type")
	}
}

// message structs
type messageHop struct {
	prevNode model.NodeId
	time     float64
}

func (mh messageHop) String() string {
	return fmt.Sprintf("n%v@%f", mh.prevNode, mh.time)
}

// a message
type Message struct {

	// a unique, unchanging message ID
	MessageId string `json:"id"`

	// sender
	Sender model.NodeId `json:"sender"`

	// message type
	Type AddressType `json:"type"`

	// The intended destination.  If it's Unicast, then this should be castable
	// to a NodeId. If it's multicast, then to a list of NodeIds.
	Destination     interface{}  `json:"destination"`
	DestinationNode model.NodeId `json:"destinationnode"`
	Payload         interface{}  `json:"payload"`

	//popularity flags - negative means it is not
	//popular in this means. The MPop is for the
	//most popular, the Lpop for the least.
	MPop int `gorm:"type:tinyint"`
	LPop int `gorm:"type:tinyint"`
	// the time the message was originated
	CreationTime float64 `json:"time"`

	// the path the message has taken so far.  This is used for simulation
	// purposes and not intended to actually be included in a message in a
	// real-world implementation.
	path []*messageHop

	//split to shards option
	ShardsAvailable int `json:"shards"`

	//ShardID for reassmble
	ShardID int `json:"shardid"`
	//more shards to go
	MShards bool `json:"ms"`

	//fake message boolean - default is false
	FakeMessage bool

	//ttl values
	TTLHops int `json:"ttl_hops"`
	TTLSecs int `json:"ttl_secs"`
	LatHops int `json:"lathops"`

	//message size (in kb)
	Size float32 `json:"size"`
}

// delivered messages
var DeliveredMessagesQueue *model.FastQueue

func (m Message) String() string {
	return fmt.Sprintf("[message] payload=%v id=%v type=%v from=%v to=%v created=%v path=%v shard=%v shardid=%v moreshards=%v fake=%v top_popular=%v least_popular=%v ttl_hops=%v ttl_secs=%v hops_passed=%v size=%v destinationNode=%v",
		m.Payload,
		m.MessageId,
		addressTypeToString(m.Type),
		m.Sender,
		m.Destination,
		m.CreationTime,
		m.path,
		m.ShardsAvailable,
		m.ShardID,
		m.MShards,
		m.FakeMessage,
		m.MPop,
		m.LPop,
		m.TTLHops,
		m.TTLSecs,
		m.LatHops,
		m.Size,
		m.DestinationNode)
}

// this function checks if a message was delievered
func (m Message) IsDeliveredYet() bool {
	return DeliveredMessagesQueue.Contains(m.MessageId)
}

// this function update the message queue that a message delievered
func (m Message) MessageDelivered() {
	DeliveredMessagesQueue.Enqueue(m.MessageId)
}

// get the destination of the message as a string
func (m Message) GetDestinationString() string {
	switch m.Destination.(type) {
	case [3]float64:
		des := m.Destination.([3]float64)
		str_des := fmt.Sprintf("%.2f,%.2f,%.2f", des[0], des[1], des[2])
		return str_des
	case model.NodeId:
		return model.NodeIdString(m.Destination.(model.NodeId))
	}
	return ""
}

// get a string that describes path of the message
func (m Message) GetPathString() string {
	full_path := ""
	for _, hop := range m.path {
		nodeId := int(hop.prevNode)
		s := strconv.Itoa(nodeId)
		// add the id and time to string
		full_path = full_path + s + ":" + strconv.FormatFloat(hop.time, 'E', -1, 64) + ","

	}
	return full_path
}

func (m *Message) Copy() *Message {
	newMessage := &Message{
		MessageId:       m.MessageId,
		Sender:          m.Sender,
		Type:            m.Type,
		Destination:     m.Destination,
		Payload:         m.Payload,
		CreationTime:    m.CreationTime,
		ShardsAvailable: m.ShardsAvailable,
		ShardID:         m.ShardID,
		MShards:         m.MShards,
		FakeMessage:     m.FakeMessage,
		MPop:            m.MPop,
		LPop:            m.LPop,
		TTLHops:         m.TTLHops,
		TTLSecs:         m.TTLSecs,
		Size:            m.Size,
		LatHops:         m.LatHops,
		DestinationNode: m.DestinationNode,
	}
	for _, hop := range m.path {
		newMessage.RecordHop(hop.prevNode, hop.time)
	}
	return newMessage
}

func (m *Message) RecordHop(prevNode model.NodeId, t float64) {
	m.path = append(m.path, &messageHop{
		prevNode: prevNode,
		time:     t,
	})
}

// reads a JSON-encoded file that describes a set of messages
// and generate random messages between nodes
// the nature of the messages' destinations will be based on the
// logic that is used in the simulator.
func LoadMessages(filename string, log *logger.Logger, logic Logic) (map[model.NodeId][]*Message, error) {
	// load messages
	dat, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// the format of the JSON file is an array of Message objects (see `Message`
	// above)
	var messages map[model.NodeId][]*Message
	if err = json.Unmarshal(dat, &messages); err != nil {
		log.Warn(err)
		return nil, err
	}
	//update the destinations as nodeid,
	//and not regular strings
	for _, node_messages := range messages {
		for _, m := range node_messages {
			m.Destination = model.NodeId(m.DestinationNode)
		}
	}
	// change the address if needed (addressing is in place)
	// and also if attackers are in place
	address_flag := false
	attacker_flag := false
	if _, ok := logic.(*Adversary); ok {
		attacker_flag = true
		if _, ok := logic.(*Adversary).L.(*AddressingLogic); ok {
			address_flag = true
		}
	}
	if _, ok := logic.(*AddressingLogic); ok {
		address_flag = true
	}
	//remove messages from and to attacker
	if attacker_flag {
		attackers := logic.(*Adversary).Adversaries
		for node, node_messages := range messages {
			_, exists := attackers[node]
			if exists { //found an attacker
				log.Infof("attacker %v was removed from messages data", node)
				delete(messages, node)
			}
			//go over destinations
			for index, m := range node_messages {
				//found a message to attacker
				//delete it
				_, exists := attackers[m.DestinationNode]
				if exists { //found an attacker
					log.Infof("message to attacker %v was removed from messages data", m.DestinationNode)
					node_messages = append(node_messages[:index], node_messages[index+1:]...)
				}
			}
		}
	}
	//if there is a need of changing the address, do it
	if address_flag {
		for _, node_messages := range messages {
			for _, m := range node_messages {
				//translate the desination node
				//to a geolocation
				pop, _ := Profilier.Popularity.Load(m.DestinationNode)
				populars := pop.(NodesPopular)
				mostP := populars.MostPopular[0]
				m.Destination = mostP
			}
		}
	}

	return messages, nil
}

// Record messages.  This function should be started as a goroutine.  It waits
// for incoming messages and records them in the database, in batches for
// efficiency.
func RecordMessages(experimentName string, messageDBChan chan *model.MessageDB, barrier *sync.WaitGroup) {
	const batchsize = 100 // an arbitrary choice
	messages := make([]*model.MessageDB, 0, batchsize)

	for mes := range messageDBChan {

		// add this encounter to our list of encounters
		messages = append(messages, mes)

		// if we've reached our batch size, send them to the DB
		if len(messages) >= batchsize {
			if r := model.DB.Create(&messages); r.Error != nil {
				logg.Warnf("failed to record messages: %v", r.Error)
			}
			messages = nil // reset the buffer
		}
	}

	// if we get here, that means that the encounterChan has been closed.

	// do we have any left over?
	if len(messages) > 0 {
		// if we have any left over in the queue, flush them to the DB
		if r := model.DB.Create(&messages); r.Error != nil {
			logg.Warnf("failed to record messages: %v", r.Error)
		}
	}
	barrier.Done()
}

// Record delivered messages.  This function should be started as a goroutine.  It waits
// for incoming messages and records them in the database, in batches for
// efficiency.
func RecordDeliveredMessages(experimentName string, messageDBChan chan *model.DeliveredMessageDB, barrier *sync.WaitGroup) {
	const batchsize = 1000 // an arbitrary choice
	messages := make([]*model.DeliveredMessageDB, 0, batchsize)

	for mes := range messageDBChan {

		// add this encounter to our list of encounters
		messages = append(messages, mes)

		// if we've reached our batch size, send them to the DB
		if len(messages) >= batchsize {
			if r := model.DB.Create(&messages); r.Error != nil {
				logg.Warnf("failed to record delivered messages: %v", r.Error)
			}
			messages = nil // reset the buffer
		}
	}

	// if we get here, that means that the encounterChan has been closed.

	// do we have any left over?
	if len(messages) > 0 {
		// if we have any left over in the queue, flush them to the DB
		if r := model.DB.Create(&messages); r.Error != nil {
			logg.Warnf("failed to record encounters: %v", r.Error)
		}
	}
	barrier.Done()
}

// shard info struct for message reassmeble
type ShardInfo struct {
	MShards bool
	ShardId int
	Id      string
	Payload string
}

// helper function for reassemble
func containsAllNumbers(slice []int) bool {
	n := len(slice)
	set := make(map[int]bool)
	for _, value := range slice {
		set[value] = true
	}
	for i := 0; i < n; i++ {
		if !set[i] {
			return false
		}
	}
	return true
}

// reassmeble a message at destination
func ReassmbleMessage(mesmap *sync.Map, dm *model.DeliveredMessageDB, m *model.MessageDB, creationTime float64) {
	values := make([]*ShardInfo, 0) //store the shards ids
	var message *Message
	//iterate the map of messages in map
	//relate only to the new shard that arrive
	//and its relative shards
	mesmap.Range(func(k, v any) bool {
		message = v.(*Message)
		if message.ShardsAvailable == 1 && strings.Split(m.MessageId, "_")[0] == strings.Split(message.MessageId, "_")[0] {
			s := &ShardInfo{MShards: message.MShards,
				ShardId: message.ShardID,
				Id:      strings.Split(m.MessageId, "_")[0],
				Payload: message.Payload.(string)}
			values = append(values, s) //store only shards info
		}
		return true
	})
	//iterate the array to find if there is a message to concat
	for i := range values {

		//init a slice for the shards with the initiator
		tmp_shards := make(map[int]*ShardInfo, 0)
		id_shards := make([]int, 0)
		tmp_shards[values[i].ShardId] = values[i]
		id_shards = append(id_shards, values[i].ShardId)
		for j := i + 1; j < len(values); j++ {
			//shards of the same message, because the messageID is the same
			if values[i].ShardId == values[j].ShardId {
				//append the value to the tmp array
				tmp_shards[values[j].ShardId] = values[j]
				id_shards = append(id_shards, values[j].ShardId)
			}
		}
		//check if the keys describe full range of shards
		//if not, do not reassemble
		if !containsAllNumbers(id_shards) {
			continue
		}
		// Convert the map to a slice of key-value pairs.
		var pairs_shards []struct {
			Key   int
			Value *ShardInfo
		}
		for k, v := range tmp_shards {
			pairs_shards = append(pairs_shards, struct {
				Key   int
				Value *ShardInfo
			}{k, v})
		}

		// Sort the slice by its keys.
		sort.Slice(pairs_shards, func(i, j int) bool {
			return pairs_shards[i].Key < pairs_shards[j].Key
		})
		//iterate the map
		payload := ""
		//check if there are more shards to come
		//if so, do not reassmble
		if pairs_shards[len(pairs_shards)-1].Value.MShards {
			continue
		}
		//reassemble
		for _, shard := range pairs_shards {
			payload = payload + " " + shard.Value.Payload
		}
		newMessage := message.Copy()
		newMessage.Payload = payload
		newMessage.ShardsAvailable = 0 //reassembled message doe not need more shards
		//setting the time for documention
		now := time.Now()
		unixTimestamp := now.Unix()
		timestampAsFloat := float64(unixTimestamp)
		delta_time := creationTime - timestampAsFloat
		dm.DeliverTime = delta_time
		dm.Payload = payload
		//m.TransferTime = delta_time
		//m.Payload = payload

		//messageDBChan <- m                             // send the reassmebled message to document channel
		receivedmessageDBChan <- dm                    // send the reassembled message to delivered message channel
		mesmap.Store(newMessage.MessageId, newMessage) //store the assembled message
		return                                         //we reassembled, no more actions needed
	}

}

// random id generator
func GenerateRandomId(nodesamount int64) string {
	//generate random message id

	// Generate a random number in the range of 1 million

	randomNumber := model.Intn(nodesamount) + 1
	return fmt.Sprintf("%d", randomNumber)
}
