package logic

import (
	model "marathon-sim/datamodel"
	"sync"
)

// DistanceBased ideality
type DistanceBasedIdeality struct {
	EncountersMapping sync.Map
	N                 int //length of distancebased ideality
}

// check the Ideality of a message and an ideal
func (c *DistanceBasedIdeality) IdealityCheck(m *Message, nodeid *model.NodeId) bool {
	emptyMap := new(sync.Map)

	v, _ := c.EncountersMapping.LoadOrStore(nodeid, emptyMap)
	idealityNodeMap := v.(*sync.Map)
	//check if the nodeid exists in the ideality
	//map, if not - create its ideality
	//and save it to ideal var
	found := false
	idealityNodeMap.Range(func(key, value interface{}) bool {
		if value.(*model.NodeId) == nodeid {
			found = true
		}
		return true
	})
	return found
}

// update ideality if neccesary
// and return it
func (c *DistanceBasedIdeality) UpdateIdeality(m *Message, nodeid *model.NodeId) bool {

	//get the map of the node
	emptyMap := new(sync.Map)

	v, _ := c.EncountersMapping.LoadOrStore(nodeid, emptyMap)
	idealityNodeMap := v.(*sync.Map)
	//get the size of the map
	count := 0
	idealityNodeMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	//the map is full, delete the oldest nodeid
	//and put the newest as nodeid 1
	if count == c.N {
		//delete the oldest
		idealityNodeMap.Delete(count)
		//cleared one slot, decrease the count
		count--
	}
	//an erase is in need
	for i := count; i > 0; i-- {
		//update the other nodes
		cur_node, _ := idealityNodeMap.Load(i)
		idealityNodeMap.Store(i+1, cur_node)
	}
	//put the new node in place
	idealityNodeMap.Store(1, nodeid)
	//return ideality
	return true
}

// get ideality name type
func (c *DistanceBasedIdeality) GetIdeality() IdealityVar {
	return DistanceBased
}

// get ideaility string
func (c *DistanceBasedIdeality) GetIdealityString() string {
	return "DistanceBased"
}

// delete ideality of a mssage
func (c *DistanceBasedIdeality) DeleteIdeailtyMessage(m *Message, nodeid *model.NodeId) {
}
