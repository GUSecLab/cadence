package logic

import (
	"database/sql"
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type RandomWalkLogicV1 struct {
	log *logger.Logger

	//probabilistic transfer
	transferProbability float64
	deleteProbability   float64
}

// initlizing the logics with the content block and a log
// in the randomwalk case, set the probabilities
func (rwl *RandomWalkLogicV1) InitLogic(block *LogicConf, log *logger.Logger) {
	rwl.log = log
	//set the probabilities
	rwl.SetTransferProbability(float64(block.RandomwalkTransferProbability))
	rwl.SetDeleteProbability(float64(block.RandomwalkDeleteProbability))
}

// change the probability of the randomwalk logic
func (rwl *RandomWalkLogicV1) SetTransferProbability(prob float64) {
	rwl.transferProbability = prob
}

// return the probability of the randomwalk logic
func (rwl *RandomWalkLogicV1) GetTransferProbability() float64 {
	return rwl.transferProbability
}

// change the probability of the randomwalk logic
func (rwl *RandomWalkLogicV1) SetDeleteProbability(prob float64) {
	rwl.deleteProbability = prob
}

// return the probability of the randomwalk logic
func (rwl *RandomWalkLogicV1) GetDeleteProbability() float64 {
	return rwl.deleteProbability
}

// This is more of an example than a useful function.  Whenever a node updates
// its position and this function is called, NewPositionCallback will update the
// state variable `lastpos` with the current position.  This isn't really used
// anywhere, but is intended to demonstrate how state is handled.
func (rwl *RandomWalkLogicV1) NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) {

}

func (rwl *RandomWalkLogicV1) HandleHelper(encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) float32 {
	actual_bandwidth := float32(0.0)

	// iterate thru messages on node1 and add them all to node2
	messageMap1.Range(func(k, v any) bool {

		messageid := k.(string)
		message := v.(*Message)
		//check the probability transfer of this message
		if !model.TrueWithProbability(rwl.transferProbability) {
			//low probability of trasnfer, abort!
			return true
		}

		//hand over the nandling to the general logic function,
		//as it works for a couple of logics
		transfer := TransferMessage(encounter, messageMap1, messageMap2, nodeid1, nodeid2, messageid, message)
		if transfer {
			//update message conunter
			actual_bandwidth++
			//check the probability transfer of this message
			if model.TrueWithProbability(rwl.deleteProbability) {

				//delete the message
				DeleteMesNode(nodeid1, message)

			}

		}

		return true
	})

	return actual_bandwidth
}

// transfers all messages between the nodes
func (rwl *RandomWalkLogicV1) HandleEncounter(encounter *model.Encounter) float32 {

	// get the two node IDs involved
	nodeid1 := encounter.Node1
	nodeid2 := encounter.Node2
	// lock message transfering
	nodemem_erase, _ := Storage.NodesMemories.Load(nodeid1)
	nodemem_serialized_erase := nodemem_erase.(*NodeMemory)
	nodemem_mutex_erase := nodemem_serialized_erase.NodeMutex
	nodemem_mutex_erase.Lock()
	nodemem, _ := Storage.NodesMemories.Load(nodeid2)
	nodemem_serialized := nodemem.(*NodeMemory)
	nodemem_mutex := nodemem_serialized.NodeMutex
	nodemem_mutex.Lock()

	// get each node's message map
	messageMap1 := GetMessageQueue(nodeid1)
	messageMap2 := GetMessageQueue(nodeid2)

	band1 := rwl.HandleHelper(encounter, messageMap1, messageMap2, nodeid1, nodeid2)
	band2 := rwl.HandleHelper(encounter, messageMap2, messageMap1, nodeid2, nodeid1)

	//unlock message transferring
	nodemem_mutex.Unlock()
	//unlock the erase lock
	nodemem_mutex_erase.Unlock()
	//return the amount of messages transfered
	return band1 + band2
}

// data preparation
func (rwl *RandomWalkLogicV1) DataPreparation(rows *sql.Rows, node_list *sql.Rows, logg *logger.Logger, minevents int) (sync.Map, []model.NodeId) {
	return DataPreparationGeneral(rows, node_list, logg, minevents)
}

// get the logic name
func (rwl *RandomWalkLogicV1) GetLogicName() string {
	return "randomwalk-v1"
}
