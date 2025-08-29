package datamodel

import (
	"fmt"
	"sync"
)

type WhenHaveISawUCondition struct {
	name               string
	minimal_transition float64   // minimal transition time
	encounters_memory  *sync.Map // a map of time of encounters

}

func (pr WhenHaveISawUCondition) GetType() ConditionType {
	return ConditionWhenHaveISawUCondition
}

func (pr WhenHaveISawUCondition) GetName() string {
	return pr.name
}

func (pr *WhenHaveISawUCondition) String() string {
	return fmt.Sprintf("{whenHaveISawU condition; minimal transition time=%v}", pr.minimal_transition)
}

func (pr *WhenHaveISawUCondition) IsConditionMet(event1, event2 *Event, node1mes *sync.Map, node2mes *sync.Map, timestamp float64) bool {
	//check if encounter is stored
	//first, take the time
	//times of the two events are identical by design
	// (look physics.Distance function to assure that)
	var op [2]int
	//assure the two of nodes in the theoratical encounter
	//in memory, as in the simulate function
	//encounters are only when node i < node j
	if NodeIdInt(event1.Node) < NodeIdInt(event2.Node) {
		op = [2]int{NodeIdInt(event1.Node), NodeIdInt(event2.Node)}
	} else {
		op = [2]int{NodeIdInt(event2.Node), NodeIdInt(event1.Node)}
	}

	encounter_in_memory, exists := pr.encounters_memory.Load(op)

	//the two encounters collide
	//then prohibit the second encounter from happening
	if exists && ((timestamp - encounter_in_memory.(float64)) < pr.minimal_transition) {
		return false
	}

	return true
}

// update if needed,given an encounter was established
// in the last seen case, update the new encounter
func (pr *WhenHaveISawUCondition) UpdateEncounterConsequences(encounter *Encounter) {
	time := encounter.Time
	op := [2]int{NodeIdInt(encounter.Node1), NodeIdInt(encounter.Node2)}
	//the other option is that there isn't an encounter in memory
	//or that the last encounter is too far in the past.
	//either way, remember this encounter and okay
	//the new encounter
	pr.encounters_memory.Store(op, time)
}
