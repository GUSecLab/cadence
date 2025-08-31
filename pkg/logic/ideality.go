package logic

import (
	model "marathon-sim/datamodel"
)

// type of message
type IdealityVar int64

const (
	CoinFlip      IdealityVar = 0
	DistanceBased IdealityVar = 1
)

// ideality interface to determine if a node
// is idealic
type Ideality interface {
	//this function will caclualte the ideality of a node
	//per message
	IdealityCheck(m *Message, nodeid *model.NodeId) bool
	//update ideality, including
	UpdateIdeality(m *Message, nodeid *model.NodeId) bool
	//get ideality name type
	GetIdeality() IdealityVar
	//get ideaility string
	GetIdealityString() string
	//get ideaility string
	DeleteIdeailtyMessage(m *Message, nodeid *model.NodeId)
}
