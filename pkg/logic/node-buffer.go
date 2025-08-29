package logic

import (
	model "marathon-sim/datamodel"
)

type NodeBuffer interface {
	InitNodeMemory(config *model.Config, n model.NodeId)
	UpdateMemoryNode(nodeid model.NodeId, message *Message)
}