package logic

import (
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// regarding the data transfer
var Storage *Memory

// the memory structure
type Memory struct {
	NodesMemories *sync.Map
	log           *logger.Logger
	Organizer     MemoryOrganizer //organizer of memory, for memory checks
}

// memory struct
type NodeMemory struct {
	BufferSize    float32
	BufferUsage   float32
	MaxBuffer     float32
	NodeMutex     *sync.Mutex
	MessagesQueue *sync.Map
}

// init device memory control
func InitMemory(log *logger.Logger, min_buffer int) {
	Storage = new(Memory)
	Storage.NodesMemories = new(sync.Map)

	Storage.log = log
	//generate default memory size
	gen_mem := &NodeMemory{BufferSize: float32(min_buffer), BufferUsage: float32(0.0), MaxBuffer: float32(0.0)}
	//save the default buffer/memory data
	Storage.NodesMemories.Store(model.NodeId(-1), gen_mem)

}

// init node memory control
func InitNodeMemory(min_buffer int, max_buffer int, n model.NodeId) {
	tmp_buffer := int64(min_buffer) + model.Intn(int64(max_buffer-min_buffer))
	//update tmp struct for buffer/memory
	tmp_mem := &NodeMemory{BufferSize: float32(tmp_buffer), BufferUsage: 0.0, MaxBuffer: 0.0, NodeMutex: new(sync.Mutex), MessagesQueue: &sync.Map{}}
	//store the memory/buffer of the node
	Storage.NodesMemories.Store(n, tmp_mem)
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
func UpdateMemoryNode(nodeid model.NodeId, message *Message) {
	size := message.Size
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		logg.Infof("can't update the memory of node %v", nodeid)
	}
	nodemem_serialized := nodemem.(*NodeMemory)
	//check if the situation is bounded or not
	//loop until solution is found
	for !Storage.Organizer.CheckMemory(nodeid, size) {
		Storage.Organizer.MakeRoom(nodeid)
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
}

// this function returns the maximum buffer size
// of the set of nodes
func MaxBufferUsage() (float32, model.NodeId) {
	max := float32(0.0)
	node := model.NodeId(-1)
	Storage.NodesMemories.Range(func(k, v any) bool {
		current_max := v.(*NodeMemory).MaxBuffer
		if max < current_max {
			max = current_max
			node = k.(model.NodeId)
		}
		return true
	})
	return max, node
}

// copy mechanism for a nodememory
func (n *NodeMemory) Copy() *NodeMemory {
	newNodeMemory := &NodeMemory{
		BufferSize:  n.BufferSize,
		BufferUsage: n.BufferUsage,
		MaxBuffer:   n.MaxBuffer,
		NodeMutex:   n.NodeMutex,
	}
	n.MessagesQueue.Range(func(k, v any) bool {
		newNodeMemory.MessagesQueue.Store(k, v.(*Message).Copy())
		return true
	})
	return newNodeMemory
}

// delete a message from the memory
func DeleteMesNode(nodeid model.NodeId, message *Message) {
	//find the syncmap of the node
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok {
		Storage.log.Infof("non exist nodeid: %v", nodeid)
		return
	}
	nodememory := nodemem.(*NodeMemory)
	mesque := nodememory.MessagesQueue
	//update the buffer usage
	nodememory.BufferUsage -= message.Size
	//erase the message by id
	mesque.Delete(message.MessageId)
	//update the amount of messages copies
	//with the addition of 1
	ChangeMessageCount(message.MessageId, -1)
	// update the message queue
	nodememory.MessagesQueue = mesque
	Storage.NodesMemories.Store(nodeid, nodememory)
}
