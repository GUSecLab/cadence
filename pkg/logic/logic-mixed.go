package logic

import (
	"database/sql"
	model "marathon-sim/datamodel"
	"os"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type MixedLogic struct {
	log *logger.Logger

	// a map from NodeId to a map of messagesMap
	messagesMap sync.Map
	//hold a randomwalk variant for randomwalk uses
	RandomwalkVariant Logic
	//hold a flooding logic
	Flooding Logic
	//ideality variant
	IdealityVariant Ideality
	//delete probability
	deleteProbability float64
	//lock sync.map
	lockMap sync.Map
}

// initlizing the logics with the content block and a log
// in the mixed case, set the randomwalk and flooding parameters
func (rwl *MixedLogic) InitLogic(block *LogicConf, log *logger.Logger) {
	rwl.log = log
	var err error
	//flooding logic for mixed
	rwl.Flooding, err = GetLogicByName("broadcast")
	if err != nil {
		log.Info("error in creating the flood for the mixed rounting,aborting")
		os.Exit(0)
	}

	//choose randomwalk by input
	ver_rndwalk := block.MixedRandomVersion
	if ver_rndwalk == 1 {
		rwl.RandomwalkVariant, err = GetLogicByName("randomwalk-v1")
		if err != nil {
			log.Info("error in creating the randomwalk-v1 for the mixed rounting,aborting")
			os.Exit(0)
		}
		//assign probabilities of rndwalk1
		rwl.RandomwalkVariant.(*RandomWalkLogicV1).SetTransferProbability(float64(block.RandomwalkTransferProbability))
		rwl.RandomwalkVariant.(*RandomWalkLogicV1).SetDeleteProbability(float64(block.RandomwalkDeleteProbability))
	} else {
		rwl.RandomwalkVariant, err = GetLogicByName("randomwalk-v2")
		if err != nil {
			log.Info("error in creating the randomwalk-v2 for the mixed rounting,aborting")
			os.Exit(0)
		}
		//assign probabilities of rndwalk1
		rwl.RandomwalkVariant.(*RandomWalkLogicV2).SetConstant(float64(block.ConstantTransferProbability))
		rwl.RandomwalkVariant.(*RandomWalkLogicV2).SetDeleteProbability(float64(block.RandomwalkDeleteProbability))
	}
	//choose ideality
	if block.MixedIdealityVersion == 0 {
		//for a coinflip, set main coin flip probability
		i := new(CoinFlipIdeality)
		i.IdealityProbability = float64(block.IdealityCoinFlip)
		i.IdealityMapping = sync.Map{}
		rwl.IdealityVariant = i
	} else {
		//for a distance based, set N
		i := new(DistanceBasedIdeality)
		i.N = block.IdealityDistanceN
		rwl.IdealityVariant = i
	}
	//init lock map
	rwl.lockMap = sync.Map{}
}

// This is more of an example than a useful function.  Whenever a node updates
// its position and this function is called, NewPositionCallback will update the
// state variable `lastpos` with the current position.  This isn't really used
// anywhere, but is intended to demonstrate how state is handled.
func (rwl *MixedLogic) NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) {

}

// change the probability of the randomwalk logic
func (rwl *MixedLogic) SetDeleteProbability(prob float64) {
	rwl.deleteProbability = prob
}

// return the probability of the randomwalk logic
func (rwl *MixedLogic) GetDeleteProbability() float64 {
	return rwl.deleteProbability
}

func (rwl *MixedLogic) HandleHelper(config *model.Config, encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) (float32, float32) {
	actual_bandwidth := float32(0.0)
	droppedMessages := float32(0.0)

	// iterate thru messages on node1 and add them all to node2
	messageMap1.Range(func(k, v any) bool {

		message := v.(*Message)
		//store the message for tmp use
		//as a new map, so the subordinate routing
		//algorithm will use it correctly
		t_map := sync.Map{}
		t_map.Store(k, message.Copy())
		//determine if the reciever is ideal
		ideal_reciever := rwl.IdealityVariant.IdealityCheck(message, &nodeid2)
		//save the ideality type ideal bandwidth and actual bandwidth
		ideality_actual := float32(0.0)
		drops := float32(0.0)

		//ideal/non ideal reciever transferring
		if ideal_reciever {
			ideality_actual, drops = rwl.Flooding.HandleHelper(config, encounter, &t_map, messageMap2, nodeid1, nodeid2)
		} else {
			ideality_actual, drops = rwl.RandomwalkVariant.HandleHelper(config, encounter, &t_map, messageMap2, nodeid1, nodeid2)
		}

		//update message counter
		actual_bandwidth += ideality_actual
		droppedMessages += drops
		//check if the sender is ideal, and based on that
		//delete the message
		ideal_sender := rwl.IdealityVariant.IdealityCheck(message, &nodeid1)
		//if the sender is not ideal, erase it in probability of delete
		if !ideal_sender {
			if model.TrueWithProbability(rwl.deleteProbability) {
				//delete the message
				DeleteMesNode(nodeid1, message)
			}
		}

		//add here the other side of ideality
		//then use the randomwalk variant
		return true
	})

	return actual_bandwidth, droppedMessages
}

// transfers all messages between the nodes
func (rwl *MixedLogic) HandleEncounter(config *model.Config, encounter *model.Encounter) (float32, float32) {

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

	//get ideal bandwidth and actual bandwidth
	band1, dropped1 := rwl.HandleHelper(config, encounter, messageMap1, messageMap2, nodeid1, nodeid2)
	band2, dropped2 := rwl.HandleHelper(config, encounter, messageMap2, messageMap1, nodeid2, nodeid1)

	// unlock message transferring
	nodemem_mutex.Unlock()
	// unlock the erase lock
	nodemem_mutex_erase.Unlock()
	return band1 + band2, dropped1 + dropped2
}

// data preparation
func (rwl *MixedLogic) DataPreparation(rows *sql.Rows, logg *logger.Logger, minevents int, config *model.Config) (*sync.Map, []model.NodeId) {
	return DataPreparationGeneral(rows, logg, minevents)
}

// get the logic name
func (rwl *MixedLogic) GetLogicName() string {
	return "mixed"
}
