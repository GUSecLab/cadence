package datamodel

import (
	"fmt"
	"sync"
)

type DistanceCondition struct {
	name string
	d    float64 // maximum distance, in meters
}

func (dc DistanceCondition) GetType() ConditionType {
	return ConditionTypeDistanceInMeters
}

func (dc DistanceCondition) GetName() string {
	return dc.name
}

func (dc *DistanceCondition) String() string {
	return fmt.Sprintf("{distance condition; dist=%v meters}", dc.d)
}

func (dc *DistanceCondition) IsConditionMet(event1, event2 *Event, node1mes *sync.Map, node2mes *sync.Map, timestamp float64) bool {
	c1 := &Coord{
		X: event1.X,
		Y: event1.Y,
		Z: event1.Z,
	}
	c2 := &Coord{
		X: event2.X,
		Y: event2.Y,
		Z: event2.Z,
	}

	dist := EuclidDistance{}
	coordDist, err := dist.Distance(c1, c2)
	if err != nil {
		log.Panicf("distance calculation error: %v", err)
	}
	return coordDist <= dc.d
}

// update if needed,given an encounter was established
// in the distance case, no update is needed
func (dc *DistanceCondition) UpdateEncounterConsequences(encounter *Encounter) {}

// get the distance of this condition
func (dc *DistanceCondition) GetDistance() float64 {
	return dc.d
}
