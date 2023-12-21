package logic

import (
	model "marathon-sim/datamodel"
)

// this is the equivalent of a ConditionType enum
type OrganizerType int64

const (
	OrganizerTypeUnknown OrganizerType = iota
	OrganizerTypeSimpleBounded
	OrganizerTypeSimpleUnbounded
)

// memory interface - for now it is only for checking the memory
type MemoryOrganizer interface {
	//check if the memory buffer is overloaded
	CheckMemory(nodeid model.NodeId, size float32) bool
	// erase a message/messages based on differen factors
	MakeRoom(nodeid model.NodeId)
	// organizer type
	OrganizerTypeReturn() OrganizerType
}

// memory organizer profiles - for now bounded and unbounded
// bounded
type SimpleBoundedOrganizer struct{}

// return the organizer type
func (or *SimpleBoundedOrganizer) OrganizerTypeReturn() OrganizerType {
	return OrganizerTypeSimpleBounded
}

// checking the memory by a bounded buffer
func (or *SimpleBoundedOrganizer) CheckMemory(nodeid model.NodeId, size float32) bool {

	nodemem, _ := Storage.NodesMemories.Load(nodeid)
	nodemem_serialized := nodemem.(*NodeMemory)
	//compare buffer size and message size
	return (size + nodemem_serialized.BufferUsage) <= nodemem_serialized.BufferSize

}

// making room by a simple bounded techinque
// it erases the oldest message
func (or *SimpleBoundedOrganizer) MakeRoom(nodeid model.NodeId) {
	nodemem, ok := Storage.NodesMemories.Load(nodeid)
	if !ok { // a problem getting the node memory
		Storage.log.Infof("can't get the node %v memory", nodeid)
	}
	//get the messages queue
	messages := nodemem.(*NodeMemory).MessagesQueue
	// initialize oldestMessage to be nil
	var oldestMessage *Message
	oldestMessage = nil

	// iterate over the sync.Map and find the oldest message
	messages.Range(func(key, value interface{}) bool {
		// convert the value to a Message
		message := value.(*Message)

		// if oldestMessage is nil or the current message is older than the oldest message, update oldestMessage
		if oldestMessage == nil || message.CreationTime < oldestMessage.CreationTime {
			oldestMessage = message
		}
		return true
	})
	//update usage decrease
	nodemem.(*NodeMemory).BufferUsage -= oldestMessage.Size
	//delete the message from the message queue
	messages.Delete(oldestMessage.MessageId)
	//update the memory usage in general and for the node
	nodemem.(*NodeMemory).MessagesQueue = messages
	//update the storage control
	Storage.NodesMemories.Store(nodeid, nodemem)

}

// unbounded
type UnboundedOrganizer struct{}

// return the organizer type
func (or *UnboundedOrganizer) OrganizerTypeReturn() OrganizerType {
	return OrganizerTypeSimpleUnbounded
}

// return true, as unbounded does not care about the max bound
func (or *UnboundedOrganizer) CheckMemory(nodeid model.NodeId, size float32) bool {
	return true

}

// in the case of unbounded buffer, there is no need for this
func (or *UnboundedOrganizer) MakeRoom(nodeid model.NodeId) {
}
