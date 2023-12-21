package datamodel

import "sync"

type NullCondition struct {
	name string
}

func (nc NullCondition) GetType() ConditionType {
	return ConditionTypeNull
}

func (nc NullCondition) GetName() string {
	return nc.name
}

func (nc *NullCondition) String() string {
	return "{null condition}"
}

func (nc *NullCondition) IsConditionMet(event1, event2 *Event, node1mes *sync.Map, node2mes *sync.Map, timestamp float64) bool {
	return true
}

// update if needed,given an encounter was established
// in the probability case, no update is needed
func (nc *NullCondition) UpdateEncounterConsequences(encounter *Encounter) {}
