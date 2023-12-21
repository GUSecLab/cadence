package datamodel

import (
	"fmt"
	"sync"
)

type ProbabilityCondition struct {
	name string
	p    float64 // probability for sending a message
}

// return true with probaility p
func TrueWithProbability(p float64) bool {
	return Float64() <= p
	//return Float64() <= p
}

func (pr ProbabilityCondition) GetType() ConditionType {
	return ConditionTypeProbailisticMessageSend
}

func (pr ProbabilityCondition) GetName() string {
	return pr.name
}

func (pr *ProbabilityCondition) String() string {
	return fmt.Sprintf("{probability condition; prob=%v}", pr.p)
}

func (pr *ProbabilityCondition) IsConditionMet(event1, event2 *Event, node1mes *sync.Map, node2mes *sync.Map, timestamp float64) bool {

	return TrueWithProbability(pr.p)
}

// update if needed,given an encounter was established
// in the probability case, no update is needed
func (pr *ProbabilityCondition) UpdateEncounterConsequences(encounter *Encounter) {}
