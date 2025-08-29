package logic

import (
	model "marathon-sim/datamodel"
	"sync"
)

// coinflip ideality
type CoinFlipIdeality struct {
	IdealityMapping     sync.Map
	IdealityProbability float64
}

// check the Ideality of a message and an ideal
func (c CoinFlipIdeality) IdealityCheck(m *Message, nodeid *model.NodeId) bool {
	emptyMap := new(sync.Map)

	v, _ := c.IdealityMapping.LoadOrStore(nodeid, emptyMap)
	idealityNodeMap := v.(*sync.Map)
	//check if the message exists in the ideality
	//map, if not - create its ideality
	//and save it to ideal var
	if value, ok := idealityNodeMap.Load(m); !ok {
		return c.UpdateIdeality(m, nodeid)
	} else {
		return value.(bool)
	}
}

// delete ideality of a mssage
func (c CoinFlipIdeality) DeleteIdeailtyMessage(m *Message, nodeid *model.NodeId) {
	emptyMap := new(sync.Map)

	v, _ := c.IdealityMapping.LoadOrStore(nodeid, emptyMap)
	idealityNodeMap := v.(*sync.Map)
	idealityNodeMap.Delete(m)
	//update the ideality map
	c.IdealityMapping.Store(nodeid, emptyMap)
}

// update ideality if neccesary
// and return it
func (c CoinFlipIdeality) UpdateIdeality(m *Message, nodeid *model.NodeId) bool {
	emptyMap := new(sync.Map)

	v, _ := c.IdealityMapping.LoadOrStore(nodeid, emptyMap)
	idMap := v.(*sync.Map)
	ideal := false
	//check the probability transfer of this message
	if model.TrueWithProbability(c.IdealityProbability) {
		ideal = true

	}
	//store the ideality for later
	idMap.Store(m, ideal)
	c.IdealityMapping.Store(nodeid, idMap)
	//return ideality
	return ideal
}

// get ideality name type
func (c CoinFlipIdeality) GetIdeality() IdealityVar {
	return CoinFlip
}

// get ideaility string
func (c CoinFlipIdeality) GetIdealityString() string {
	return "CoinFlip"
}
