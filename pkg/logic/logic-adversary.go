//Don't forget to take care of the flooder's bandwidth!!!!

package logic

import (
	"database/sql"
	"encoding/json"
	model "marathon-sim/datamodel"
	"os"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type Adversary struct {
	log         *logger.Logger
	L           Logic                     // logic for the secure messaging
	Adversaries map[model.NodeId]Approach //list of adversaries
}

// init the logic of adversary and regular clients
func (a *Adversary) Init(log *logger.Logger) {
	a.log = log

}

// initlizing the logics with the content block and a log
// in the adversary case, set the attackers by the adversaries file
func (a *Adversary) InitLogic(block *LogicConf, log *logger.Logger) {
	//init logger
	a.log = log
	//first-read attackers definitions
	attackers, err := a.ReadUsersFile(block.AdversariesFile)
	if err != nil {
		log.Fatalf("cannot read attackers file (%v): %v", block.AdversariesFile, err)
		return
	}
	//then, set the underlyning logic
	logic, err := GetLogicByName(block.UnderlyningLogic)
	if err != nil {
		log.Fatalf("cannot retrieve logic engine with name %v: %v", logic, err)
		return
	}
	//update sub-logic of adversary
	a.L = logic
	//then,get the list of nodes to set the attacker in place
	node_lists, err := model.DB.Table("node_lists").Select("node").Where("dataset_name=?", block.Dataset).Rows()
	if err != nil {
		log.Infof("error in fetching the events for profiling")
		return
	}
	nodes := make([]model.NodeId, 0)
	for node_lists.Next() {
		var tmp_node model.Event
		if err := node_lists.Scan(&tmp_node); err != nil {
			log.Warnf("invalid node %v", err)
			continue
		}
	}
	//create the adversarial nodes
	attackers_list := a.GetRandomAttackers(len(nodes), attackers, nodes)

	//update the adversaries in place
	a.UpdateAttackerList(attackers_list)

}

// This is more of an example than a useful function.  Whenever a node updates
// its position and this function is called, NewPositionCallback will update the
// state variable `lastpos` with the current position.  This isn't really used
// anywhere, but is intended to demonstrate how state is handled.
func (a *Adversary) NewPositionCallback(nodeid model.NodeId, t model.LocationType, b []byte) {
	a.L.NewPositionCallback(nodeid, t, b)
}

// transfers all messages between the nodes
func (a *Adversary) HandleEncounter(encounter *model.Encounter) float32 {
	// get the two node IDs involved
	nodeid1 := encounter.Node1
	nodeid2 := encounter.Node2
	// get each node's message map
	messageMap1 := GetMessageQueue(nodeid1)
	messageMap2 := GetMessageQueue(nodeid2)

	//check if the first node is an attacker
	//and the second is not
	//end if both are attackers
	if _, ok := a.Adversaries[nodeid1]; ok {
		if _, ok := a.Adversaries[nodeid2]; !ok {
			//check if the second node is an attacker - ?
			// the attacker works
			band2 := a.L.HandleHelper(encounter, messageMap2, messageMap1, nodeid2, nodeid1)
			band1 := a.HandleHelper(encounter, messageMap1, messageMap2, nodeid1, nodeid2)
			//return the
			return band1 + band2
		}
		//two adversaries, not bandwith used
		return 0
	}
	//check if the second node is an attacker
	//and the first is not
	//end if both are attackers
	if _, ok := a.Adversaries[nodeid2]; ok {
		if _, ok := a.Adversaries[nodeid1]; !ok {
			//check if the second node is an attacker - ?
			// the attacker works
			band2 := a.L.HandleHelper(encounter, messageMap1, messageMap2, nodeid1, nodeid2)
			band1 := a.HandleHelper(encounter, messageMap2, messageMap1, nodeid2, nodeid1)
			return band1 + band2
		}
		//two adversaries - no bandwith used
		return 0
	}
	//otherwise, run the suboridnate logic mechnisem
	//and get the amount of messages transfered
	return a.L.HandleEncounter(encounter)

}

// update the list of attackers
func (a *Adversary) UpdateAttackerList(attackers map[model.NodeId]Approach) {
	a.Adversaries = attackers
}

// data preparation
func (a *Adversary) DataPreparation(rows *sql.Rows, node_list *sql.Rows, logg *logger.Logger, minevents int) (sync.Map, []model.NodeId) {
	return DataPreparationGeneral(rows, node_list, logg, minevents)
}

// handle attacker function
// this function calls the actual attack on the regular node
func (a *Adversary) HandleHelper(encounter *model.Encounter, attackerMap *sync.Map, targetMap *sync.Map, nodeattacker model.NodeId, regularnode model.NodeId) float32 {
	// iterate thru messages on attacker,
	//create fake ones
	//and return them to the client.
	//this achieves the attacker's goal -
	//to flood the net with fake messages
	// Add a key-value pair
	a.Adversaries[nodeattacker].OneSidedAttack(encounter, nodeattacker, attackerMap)
	actual_bandwidth := a.L.HandleHelper(encounter, attackerMap, targetMap, nodeattacker, regularnode)

	//delete the data for the sync map of the attacker.
	//there is no need in these fake messages in the future
	attackerMap = &sync.Map{}
	return actual_bandwidth
}

// set the attackers
func (at *Adversary) GetRandomAttackers(total int, users []*UsersInfo, nodes []model.NodeId) map[model.NodeId]Approach {
	var amount int = 0                   //amount of users
	m := make(map[model.NodeId]Approach) //map of attackers
	var t Approach                       //type of attacker
	//shuffle the nodes so that the attackers
	//are picked randomly
	// initialize the random number generator with the current time

	// shuffle the slice

	var data []interface{}
	for i := range nodes {
		data = append(data, &nodes[i])
	}

	// Shuffle the slice using math/rand
	model.Shuffle(data)
	var myStructs []model.NodeId
	for _, v := range data {
		if s, ok := v.(*model.NodeId); ok {
			myStructs = append(myStructs, *s)
		}
	}

	//indices for the attackers loop
	index_of_attackers := 0
	for _, a := range users {
		utype := a.Type //attacker type
		//get the amount of users
		amount = amount + int(float64(total)*a.Amount)
		//choose the right type
		switch utype {
		case "dropper":
			t = &DropperAttacker{}
		case "flooder":
			t = &FlooderAttacker{}

		}
		t.Init(at.log)
		// Generate random node numbers and assign the attacker approach
		for i := index_of_attackers; i < amount; i++ {
			//update the new attacer in attackers' list
			m[myStructs[i]] = t
			//update the counters of attackers
			index_of_attackers++
		}
	}
	return m
}

// users types for all logics
type UsersInfo struct {
	Type   string  `json:"type"`
	Amount float64 `json:"amount"`
}

// read attackers file
func (a *Adversary) ReadUsersFile(usersFile string) ([]*UsersInfo, error) {
	// load attackers file
	dat, err := os.ReadFile(usersFile)
	if err != nil {
		return nil, err
	}

	// the format of the JSON file is an array of Message objects (see `Message`
	// above)
	var attackers []*UsersInfo
	if err = json.Unmarshal(dat, &attackers); err != nil {
		a.log.Infof("error in marshleeing arrackers: %v", err)
		return nil, err
	}
	return attackers, nil
}

// get the logic name
func (a *Adversary) GetLogicName() string {
	return "adversary"
}
