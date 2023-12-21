package main

import (
	model "marathon-sim/datamodel"
	"math"
	"sync"
)

func updateClosestEncounterTime(prev, now float64, timeOfClosestEncounter float64) float64 {
	// Calculate the absolute differences between prev and timeOfClosestEncounter, and now and timeOfClosestEncounter
	diffPrev := math.Abs(prev - timeOfClosestEncounter)
	diffNow := math.Abs(now - timeOfClosestEncounter)

	// return timeOfClosestEncounter based on the closest value
	if diffPrev < diffNow {
		return prev
	}
	return now

}

// Figures out whether the two nodes who own `n1events` and `n2events`,
// respectively, crossed paths during this epoch.  Returns a bool indicating
// whether they crossed paths (really, whether all of the passed `conditions`
// have been met), and if so, the time at which they did so
func didCrossPaths(epoch *Epoch, n1events, n2events *startEndEvents, conditions model.EncounterConditions, node1messages *sync.Map, node2messages *sync.Map) (bool, float64, float32, float32, float32) {
	var a, b, c, d, e, f, g, h, i, j, k, l float64

	// let's have better variable names
	tStart := epoch.prev
	tEnd := epoch.now
	tDelta := tEnd - tStart

	// figure out the distance per unit time in the X dimension
	a = (n1events.end.X - n1events.start.X) / (n1events.end.Time - n1events.start.Time)
	// next, figure out where the node will be at the start of the epoch
	b = n1events.start.X + (a * (tStart - n1events.start.Time))

	// next, do the same for the Y dimension
	c = (n1events.end.Y - n1events.start.Y) / (n1events.end.Time - n1events.start.Time)
	// next, figure out where the node will be at the start of the epoch
	d = n1events.start.Y + (c * (tStart - n1events.start.Time))

	// and finally, the Z dimension
	i = (n1events.end.Z - n1events.start.Z) / (n1events.end.Time - n1events.start.Time)
	j = n1events.start.Z + (i * (tStart - n1events.start.Time))

	// and let's the same as the above, but now for the second node
	e = (n2events.end.X - n2events.start.X) / (n2events.end.Time - n2events.start.Time)
	f = n2events.start.X + (e * (tStart - n2events.start.Time))
	g = (n2events.end.Y - n2events.start.Y) / (n2events.end.Time - n2events.start.Time)
	h = n2events.start.Y + (g * (tStart - n2events.start.Time))
	k = (n2events.end.Z - n2events.start.Z) / (n2events.end.Time - n2events.start.Time)
	l = n2events.start.Z + (k * (tStart - n2events.start.Time))

	// ok, next lets figure out the time at which the nodes are closest
	alpha := a - e
	beta := c - g
	gamma := i - k

	delta1 := b - f
	delta2 := d - h
	delta3 := j - l

	num := (alpha * delta1) + (beta * delta2) + (gamma * delta3)
	denom := (alpha * alpha) + (beta * beta) + (gamma * gamma)

	var timeOfClosestEncounter float64
	if denom == 0 {
		timeOfClosestEncounter = 0.0
	} else {
		timeOfClosestEncounter = math.Min(tDelta, math.Max(0, -1.0*num/denom))
	}
	//log.Debugf("time of closest encounter is %v", timeOfClosestEncounter)

	// figure out of distance is close.  We need to create two events that occur
	// at the time of the closest passing, and then run all conditions thru it
	allConditionsPassed := true
	event1 := &model.Event{
		Time: timeOfClosestEncounter,
		Node: n1events.start.Node,
		X:    (a * timeOfClosestEncounter) + b,
		Y:    (c * timeOfClosestEncounter) + d,
		Z:    (i * timeOfClosestEncounter) + j,
	}
	event2 := &model.Event{
		Time: timeOfClosestEncounter,
		Node: n2events.start.Node,
		X:    (e * timeOfClosestEncounter) + f,
		Y:    (g * timeOfClosestEncounter) + h,
		Z:    (k * timeOfClosestEncounter) + l,
	}

	for _, condition := range conditions {
		isMet := condition.IsConditionMet(event1, event2, node1messages, node2messages, timeOfClosestEncounter+tStart)

		if !isMet { //one of the conditions was not met,abort
			return false, 0, 0, 0, 0
		}
	}

	var avX, avY, avZ float32
	avX = float32(math.Abs(event1.X+event2.X) / 2)
	avY = float32(math.Abs(event1.Y+event2.Y) / 2)
	avZ = float32(math.Abs(event1.Z+event2.Z) / 2)
	timeOfClosestEncounter = updateClosestEncounterTime(tStart, tEnd, timeOfClosestEncounter+tStart)
	//timeOfClosestEncounter = timeOfClosestEncounter + tStart
	//calculate the new encounter point
	//in which the nearest

	return allConditionsPassed, timeOfClosestEncounter, avX, avY, avZ
}
