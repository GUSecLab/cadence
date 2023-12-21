package datamodel

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sync"
)

type MessageMonitor struct {
	StatusMap *sync.Map //message statuc map
}

var MesStatusMonitor *MessageMonitor

// init the monitor of messages
func MonitorInit() {
	MesStatusMonitor = new(MessageMonitor)
	MesStatusMonitor.StatusMap = &sync.Map{}
}

// calculate the hash function of a syncmap
func calculateHashNode(m *sync.Map) []byte {
	hasher := sha256.New()

	m.Range(func(key, value interface{}) bool {
		// Convert key values only - it is the message ids
		//for fast track of the messages status
		keyBytes := []byte(fmt.Sprintf("%v", key))

		// Write key and value bytes to the hasher
		hasher.Write(keyBytes)

		return true
	})

	return hasher.Sum(nil)
}

// update a status of messages per node
func (mr *MessageMonitor) UpdateNodeMessageStatus(node1 NodeId, messages1 *sync.Map, node2 NodeId, messages2 *sync.Map) {
	//calculate hash
	h_value1 := calculateHashNode(messages1)
	h_value2 := calculateHashNode(messages2)
	//update it in the status map
	MesStatusMonitor.StatusMap.Store([2]NodeId{node1, node2}, ConcatenateBytes(h_value1, h_value2))
}

// compare two statuses
func (mr *MessageMonitor) CompareCurPastStatuses(node1 NodeId, messages1 *sync.Map, node2 NodeId, messages2 *sync.Map) bool {
	//get the past hash of the node's messages
	if past_hash, ok := MesStatusMonitor.StatusMap.Load([2]NodeId{node1, node2}); ok {
		//if there is a hash,compare the values
		h_value1 := calculateHashNode(messages1)
		h_value2 := calculateHashNode(messages2)
		h := ConcatenateBytes(h_value1, h_value2)
		res := bytes.Equal(past_hash.([]byte), h)
		return res
	}
	//return false, as there is no past hash value
	return false
}

// concanate two slices of []byte
func ConcatenateBytes(slice1, slice2 []byte) []byte {
	return append(slice1, slice2...)
}
