package logic

import (
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// regarding the data transfer
var Storage *Memory
var buffer_type string

// the memory structure
type Memory struct {
	NodesMemories *sync.Map
	log           *logger.Logger
	Organizer     MemoryOrganizer //organizer of memory, for memory checks
}

// var Buffer *NodeBuffer


// SIMPLE BUFFER TYPE 

// this has been moved to simple-buffer.go

// memory struct
type SimpleBuffer struct {
	BufferSize    float32
	BufferUsage   float32
	MaxBuffer     float32
	NodeMutex     *sync.Mutex
	MessagesQueue *sync.Map
}

// PER DEVICE BUFFER TYPE 

type PerDeviceBuffer struct {
	BufferSize    float32
}



// init device memory control
func InitMemory(log *logger.Logger, config *model.Config) {

	min_buffer := config.Simulation.MinBufferSize
	Storage = new(Memory)
	Storage.NodesMemories = new(sync.Map)

	Storage.log = log

	// TODO: THIS IS PRETTY JANKY

	// generate the specified buffer type 
	//generate default memory size
	// by default we have simple buffer type 
	buffer_type = config.Simulation.BufferType

	if buffer_type == "simple" {
		gen_mem := &SimpleBuffer{BufferSize: float32(min_buffer), BufferUsage: float32(0.0), MaxBuffer: float32(0.0)}
		Storage.NodesMemories.Store(model.NodeId(-1), gen_mem)
	}

	if buffer_type == "per-device" {
		gen_mem := &PerDeviceBuffer{BufferSize: float32(min_buffer)}
		Storage.NodesMemories.Store(model.NodeId(-1), gen_mem)
	}
}


// this should still be here, but which specific node buffer type we 
// create should be overloaded

// this code is for simple buffer only, has been moved to simple buffer

// init node memory control
func InitNodeMemory(config *model.Config, n model.NodeId) {

	min_buffer := config.Simulation.MinBufferSize;
	max_buffer := config.Simulation.MaxBufferSize;

	// temporary 
	buffer_type = config.Simulation.BufferType
	tmp_buffer := int64(min_buffer) + model.Intn(int64(max_buffer-min_buffer))

	//update tmp struct for buffer/memory
	//tmp_mem := &SimpleBuffer{BufferSize: float32(tmp_buffer), BufferUsage: 0.0, MaxBuffer: 0.0, NodeMutex: new(sync.Mutex), MessagesQueue: &sync.Map{}}
	//store the memory/buffer of the node
	if buffer_type == "simple" {
		tmp_mem := &SimpleBuffer{BufferSize: float32(tmp_buffer), BufferUsage: 0.0, MaxBuffer: 0.0, NodeMutex: new(sync.Mutex), MessagesQueue: &sync.Map{}}
		Storage.NodesMemories.Store(n, tmp_mem)
	} 
	if buffer_type == "per-device" {
		tmp_mem := &PerDeviceBuffer{BufferSize: float32(tmp_buffer)}
		Storage.NodesMemories.Store(n, tmp_mem)
	}
}


// set the organizer
func SetOrganizer(organizer OrganizerType) {
	// Using a switch statement to pick the corresponding organizer
	switch organizer {
	case 1:
		Storage.Organizer = new(SimpleBoundedOrganizer)
	case 2:
		Storage.Organizer = new(UnboundedOrganizer)
	}
}

// this function update the buffer usage if the
// message was accepted
// it also assigns the updated messages queue

// this function decides which buffer-type specific function to call
// based on the buffer type
func UpdateMemoryNode(config *model.Config, nodeid model.NodeId, message *Message) bool {
	if buffer_type == "simple" {
		return UpdateSimpleBuffer(config, nodeid, message)
	}
	if buffer_type == "per-device" {
		return UpdatePerDeviceBuffer(nodeid, message)
	}
	return false
}

// this function updates the buffer usage for simple buffer type
func UpdateSimpleBuffer(config *model.Config, nodeid model.NodeId, message *Message) bool {
	size := message.Size
	didDrop := false
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		logg.Infof("can't update the memory of node %v", nodeid)
		InitNodeMemory(config, nodeid)
	}
	nodemem_serialized := nodemem.(*SimpleBuffer)
	//check if the situation is bounded or not
	//loop until solution is found
	for !Storage.Organizer.CheckMemory(nodeid, size) {
		Storage.Organizer.MakeRoom(nodeid)
		didDrop = true
	}
	nodemem_serialized.BufferUsage += size
	//update the amount of messages copies
	//with the addition of 1
	ChangeMessageCount(message.MessageId, 1)
	//update max buffer usage over time
	if nodemem_serialized.MaxBuffer < nodemem_serialized.BufferUsage {
		nodemem_serialized.MaxBuffer = nodemem_serialized.BufferUsage
	}
	//update the message queue
	nodemem_serialized.MessagesQueue.Store(message.MessageId, message)
	Storage.NodesMemories.Store(nodeid, nodemem_serialized)

	return didDrop // return true if the message was accepted, false if it was dropped
}

func GetBufferUsage(nodeid model.NodeId) float32 {
	NodeBuffer := GetSimpleBuffer(nodeid)
	if NodeBuffer == nil {
		Storage.log.Infof("cannot get buffer usage for node: %v", nodeid)
		return 0.0
	}
	return NodeBuffer.BufferUsage
}


// this function updates the buffer usage for per-device buffer type 
func UpdatePerDeviceBuffer(nodeid model.NodeId, message *Message) bool {
	return false // TODO: implement this function
}

func GetSimpleBuffer(nodeid model.NodeId) *SimpleBuffer {
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		Storage.log.Infof("cannot get simple buffer for node: %v", nodeid)
		return nil
	}
	SimpleBuffer := nodemem.(*SimpleBuffer)
	return SimpleBuffer
}

func GetPerDeviceBuffer(nodeid model.NodeId) *PerDeviceBuffer {
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		Storage.log.Infof("cannot get per-device buffer for node: %v", nodeid)
		return nil 
	}
	PerDeviceBuffer := nodemem.(*PerDeviceBuffer)
	return PerDeviceBuffer
}



// this function returns the maximum buffer size
// of the set of nodes
func MaxBufferUsage() (float32, model.NodeId) {
	max := float32(0.0)
	node := model.NodeId(-1)
	Storage.NodesMemories.Range(func(k, v any) bool {
		current_max := v.(*SimpleBuffer).MaxBuffer
		if max < current_max {
			max = current_max
			node = k.(model.NodeId)
		}
		return true
	})
	return max, node
}

// copy mechanism for a simple type nodememory
func (n *SimpleBuffer) Copy() *SimpleBuffer {
	newSimpleBuffer := &SimpleBuffer{
		BufferSize:  n.BufferSize,
		BufferUsage: n.BufferUsage,
		MaxBuffer:   n.MaxBuffer,
		NodeMutex:   n.NodeMutex,
	}
	n.MessagesQueue.Range(func(k, v any) bool {
		newSimpleBuffer.MessagesQueue.Store(k, v.(*Message).Copy())
		return true
	})
	return newSimpleBuffer
}

func (n *PerDeviceBuffer) Copy() *PerDeviceBuffer {
	newPerDeviceBuffer := &PerDeviceBuffer{
		BufferSize:  n.BufferSize,
	}
	/*

	n.MessagesQueue.Range(func(k, v any) bool {
		newSimpleBuffer.MessagesQueue.Store(k, v.(*Message).Copy())
		return true
	})
	*/
	return newPerDeviceBuffer
}

// delete a message from the memory
// this function determines which buffer type specific function to call

func DeleteMesNode(nodeid model.NodeId, message *Message) {
	if buffer_type == "simple" {
		DeleteMesNodeSimpleBuffer(nodeid, message)
	}
	if buffer_type == "per-device" {
		DeleteMesNodePerDeviceBuffer(nodeid, message)
	}
}

// this function deletes a message from the memory for a simple buffer
func DeleteMesNodeSimpleBuffer(nodeid model.NodeId, message *Message) {
	//find the syncmap of the node
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		Storage.log.Infof("non exist nodeid: %v", nodeid)
		return
	}
	SimpleBuffer := nodemem.(*SimpleBuffer)
	mesque := SimpleBuffer.MessagesQueue
	//update the buffer usage
	SimpleBuffer.BufferUsage -= message.Size
	//erase the message by id
	mesque.Delete(message.MessageId)
	//update the amount of messages copies
	//with the addition of 1
	ChangeMessageCount(message.MessageId, -1)
	// update the message queue
	SimpleBuffer.MessagesQueue = mesque
	Storage.NodesMemories.Store(nodeid, SimpleBuffer)
}

// this function deletes a message from memory for a per-device buffer
func DeleteMesNodePerDeviceBuffer(nodeid model.NodeId, message *Message) {

}

// gets the messages map for a node based on the node ID
// calls the specific buffer type function
func GetMessageQueue(nodeid model.NodeId) *sync.Map {
	if buffer_type == "simple" {
		return GetMessageQueueSimpleBuffer(nodeid)
	}
	if buffer_type == "per-device" {
		return GetMessageQueuePerDeviceBuffer(nodeid)
	}
	return nil
}

// gets the messages map for a node based on the node ID for a simple buffer
func GetMessageQueueSimpleBuffer(nodeid model.NodeId) *sync.Map {
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		Storage.log.Infof("cannot get message queue for node: %v", nodeid)
		return nil
	}
	SimpleBuffer := nodemem.(*SimpleBuffer)
	return SimpleBuffer.MessagesQueue
}

// gets the messages map for a node based on the node ID for a per-device buffer
func GetMessageQueuePerDeviceBuffer(nodeid model.NodeId) *sync.Map {
	return nil
}


