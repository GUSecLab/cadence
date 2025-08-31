package datamodel

import (
	"sync"
)

type DoYouHaveEnoughUpdates struct {
	name string
}

func (pr DoYouHaveEnoughUpdates) GetType() ConditionType {
	return ConditionDoYouHaveEnoughUpdates
}

func (pr DoYouHaveEnoughUpdates) GetName() string {
	return pr.name
}

func (pr *DoYouHaveEnoughUpdates) String() string {
	return "{doYouHaveEnoughUpdates condition;}"
}

func (pr *DoYouHaveEnoughUpdates) IsConditionMet(event1, event2 *Event, node1mes *sync.Map, node2mes *sync.Map, timestamp float64) bool {
	//check the monitor for changes
	//if the nodes messages stay the same, return false
	//as there is no need for this encounter
	//elsewhere, return true
	return !MesStatusMonitor.CompareCurPastStatuses(event1.Node, node1mes, event2.Node, node2mes)
}

// update if needed,given an encounter was established
// in the hash case, no update is needed
func (pr *DoYouHaveEnoughUpdates) UpdateEncounterConsequences(encounter *Encounter) {}
