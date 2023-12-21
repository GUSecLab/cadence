# Logic API

## InitLogic(block *LogicConf, log *logger.Logger) - 
initialize function of a logic engine. It obtains a block that consists of hyper-parameters for initializing the logic engine, and a pointer to the logger. Each logic analyzes its parameters from the block. For example, the probability of message transfer in the Randomwalk-v1 routing algorithm.
```
The LogicConf block looks as follows:
{
    "logic": "broadcast", // name of logic engine
    "dataset_name": "napa",
    "underlyning_logic": "broadcast", //name of the underlying logic (where adversaries are in-place)
    "profile_file": "profile_defs.json", //path of profile file for humanets protocol
    "randomwalk_transfer_probability": 0.7,
    "randomwalk_delete_probability": 0,
    "randomwalk_transfer_constant": 1,
    "mixed_randomwalk_version": 1, // 0 is coin flip mixed logic, 1 is distance based
    "mixed_ideality_version": 0, // 0 is randomwalk-v1, 1 is randomwalk-v2
    "ideality_coin_flip": 0.7, //ideality probability of coin flip
    "ideality_distance_n": 1, // N factor of mixed distance based
    "organizer": 2 // component that governs the buffers, 1 is for bounded buffers, 2 for unbounded buffers
} 
```
## NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) - 
every time a node finds itself in a new position, this message is called. nodeid identifies the specific node and the model.LocationType resembles the type of location (e.g., Coord) of the node location. `b` is the marshaled form of the location, and will need to be unmarshalled by this method.  

## OriginateMessage(id model.NodeId, message *Message) - 
initially stores a message at a node (i.e., when the message is originated and first comes into existence); does NOT make a copy of that message. A message looks like:

```
{
    "id": "", //id of the message
    "sender" : 0, //sender node id
    "destination": 1, //destination node id
    "type": 10, //type of message (unicast, multicast)
    "time": 0, // creation time of the message
    "payload": "Hi ", // content of the message
    "shards": 0, //in case of a fragmented message, this flags the message as a fragment
    "shardid": 0, //in case of a fragmented message, this flags the id of the fragment
    "ms": false, //in case of a fragmented message, this flags if there are more fragments
    "ttl_hops": 20, //number of hops the message can travel
    "ttl_secs": 1000000, //number of seconds the message can live
    "size": 1 //size of the message
}
```

## HandleEncounter(encounter *model.Encounter) float32 -
 potentially does a message exchange between nodes when an encounter occurs. This function returns the bandwidth of the encounter. The input is the encounter.

## HandleHelper(encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) float32 - 
this function is exposed to set the functionality of the attackers. It allows us to define half of an encounter, in a case where an attacker is in place. The original logic engine governs the message transfer of the benign node, and the attacker logic governs what the attacker does. This function returns the bandwidth of this half of an encounter. The inputs are the encounter, the message queues and nodeids.

## GetLogicName() string - 
returns the name of the logic is string (for later comparison purposes)

## Examples of using the logic API:

### Broadcast routing engine:
```
package logic

import (
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type BroadcastLogic struct {
	log *logger.Logger
}

// initlizing the logics with the content block and a log
// in the broadcast case, nothing is needed
func (bl *BroadcastLogic) InitLogic(block *LogicConf, log *logger.Logger) {
	bl.log = log
}

// This is more of an example than a useful function.  Whenever a node updates
// its position and this function is called, NewPositionCallback will update the
// state variable `lastpos` with the current position.  This isn't really used
// anywhere, but is intended to demonstrate how state is handled.
func (bl *BroadcastLogic) NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) {

}
func (bl *BroadcastLogic) HandleHelper(encounter *model.Encounter, messageMap1 *sync.Map, messageMap2 *sync.Map, nodeid1 model.NodeId, nodeid2 model.NodeId) float32 {
	ideal_bandwidth := float32(0.0)
	// iterate thru messages on node1 and add them all to node2
	messageMap1.Range(func(k, v any) bool {
		messageid := k.(string)

		message := v.(*Message)

		//hand over the nandling to the general logic function,
		//as it works for a couple of logics
		transfer := TransferMessage(encounter, messageMap1, messageMap2, nodeid1, nodeid2, messageid, message)
		if transfer {
			//update transfer counter of messages
			ideal_bandwidth++
		}

		return true
	})

	return ideal_bandwidth
}

// transfers all messages between the nodes
func (bl *BroadcastLogic) HandleEncounter(encounter *model.Encounter) float32 {
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

	//get ideal bandwidth and actual bandwidth
	band1 := bl.HandleHelper(encounter, messageMap1, messageMap2, nodeid1, nodeid2)
	band2 := bl.HandleHelper(encounter, messageMap2, messageMap1, nodeid2, nodeid1)

	// unlock message transferring
	nodemem_mutex.Unlock()
	// unlock the erase lock
	nodemem_mutex_erase.Unlock()
	return band1 + band2
}
// get the logic name
func (bl *BroadcastLogic) GetLogicName() string {
	return "broadcast"
}
```

### Randomwalk routing engine:
```
package logic

import (
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
	ideal_bandwidth := float32(0.0)

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
			ideal_bandwidth++
			//check the probability transfer of this message
			if model.TrueWithProbability(rwl.deleteProbability) {

				//delete the message
				DeleteMesNode(nodeid1, message)

			}

		}

		return true
	})

	return ideal_bandwidth
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

// get the logic name
func (rwl *RandomWalkLogicV1) GetLogicName() string {
	return "randomwalk-v1"
}
```


