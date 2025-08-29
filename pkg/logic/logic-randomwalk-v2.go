package logic

import (
	"database/sql"
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type RandomWalkLogicV2 struct {
	log *logger.Logger

	//probabilistic transfer
	timeconstant      float64
	deleteProbability float64
}

// initlizing the logics with the content block and a log
// in the randomwalk case, set transfer constant and delete probability
func (rwl *RandomWalkLogicV2) InitLogic(block *LogicConf, log *logger.Logger) {
	rwl.log = log
	//set the probability of delete and transfer constant
	rwl.SetConstant(float64(block.ConstantTransferProbability))
	rwl.SetDeleteProbability(float64(block.RandomwalkDeleteProbability))
}

// change the probability of the randomwalk logic
func (rwl *RandomWalkLogicV2) SetConstant(consta float64) {
	rwl.timeconstant = consta
}

// return the probability of the randomwalk logic
func (rwl *RandomWalkLogicV2) GetConstant() float64 {
	return rwl.timeconstant
}

// change the probability of the randomwalk logic
func (rwl *RandomWalkLogicV2) SetDeleteProbability(prob float64) {
	rwl.deleteProbability = prob
}

// return the probability of the randomwalk logic
func (rwl *RandomWalkLogicV2) GetDeleteProbability() float64 {
	return rwl.deleteProbability
}

// This is more of an example than a useful function.  Whenever a node updates
// its position and this function is called, NewPositionCallback will update the
// state variable `lastpos` with the current position.  This isn't really used
// anywhere, but is intended to demonstrate how state is handled.
func (rwl *RandomWalkLogicV2) NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) {

}

func (rwl *RandomWalkLogicV2) HandleHelper(config *model.Config, encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) (float32, float32) {
	actual_bandwidth := float32(0.0)
	droppedMessages := float32(0.0)
	// iterate thru messages on node1 and add them all to node2
	messageMap1.Range(func(k, v any) bool {

		messageid := k.(string)
		message := v.(*Message)
		//carefully inspect this part after presentation!!!!
		//erase the outer condition and the inner p>=1
		//then add it to the model.Trueiwith....
		time_held := encounter.Time - message.CreationTime
		//check the probability transfer of this message
		p := time_held / rwl.timeconstant
		if p < 1 {
			if !(model.TrueWithProbability(p)) {
				//low probability of trasnfer, do not transfer
				return true
			}
		}

		//hand over the nandling to the general logic function,
		//as it works for a couple of logics
		transfer, didDrop := TransferMessage(config, encounter, messageMap1, messageMap2, nodeid1, nodeid2, messageid, message)
		if transfer {

			//update counter of message transfer
			actual_bandwidth++
			//check the probability transfer of this message
			if model.TrueWithProbability(rwl.deleteProbability) {

				//delete the message
				DeleteMesNode(nodeid1, message)

			}
		}

		// if the message was dropped, we need to update the memory
		if didDrop {
			droppedMessages++
		}
		//unlock message transferring
		// nodemem_mutex.Unlock()
		return true
	})
	return actual_bandwidth, droppedMessages
}

// transfers all messages between the nodes
func (rwl *RandomWalkLogicV2) HandleEncounter(config *model.Config, encounter *model.Encounter) (float32, float32) {

	// get the two node IDs involved
	nodeid1 := encounter.Node1
	nodeid2 := encounter.Node2
	// lock message transfering
	nodemem_erase, _ := Storage.NodesMemories.Load(nodeid1)
	nodemem_serialized_erase := nodemem_erase.(*SimpleBuffer)
	nodemem_mutex_erase := nodemem_serialized_erase.NodeMutex
	nodemem_mutex_erase.Lock()
	nodemem, _ := Storage.NodesMemories.Load(nodeid2)
	nodemem_serialized := nodemem.(*SimpleBuffer)
	nodemem_mutex := nodemem_serialized.NodeMutex
	nodemem_mutex.Lock()

	// get each node's message map
	messageMap1 := GetMessageQueue(nodeid1)
	messageMap2 := GetMessageQueue(nodeid2)

	band1, dropped1 := rwl.HandleHelper(config, encounter, messageMap1, messageMap2, nodeid1, nodeid2)
	band2, dropped2 := rwl.HandleHelper(config, encounter, messageMap2, messageMap1, nodeid2, nodeid1)

	//unlock message transferring
	nodemem_mutex.Unlock()
	//unlock the erase lock
	nodemem_mutex_erase.Unlock()
	//return the amount of messages transfered
	return band1 + band2, dropped1 + dropped2
}

// data preparation
func (rwl *RandomWalkLogicV2) DataPreparation(rows *sql.Rows, logg *logger.Logger, minevents int, config *model.Config) (*sync.Map, []model.NodeId) {
	return DataPreparationGeneral(rows, logg, minevents)
}

// get the logic name
func (rwl *RandomWalkLogicV2) GetLogicName() string {
	return "randomwalk-v2"
}
