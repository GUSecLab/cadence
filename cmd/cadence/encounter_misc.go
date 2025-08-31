package main

import (
	model "marathon-sim/datamodel"
	"math"
	"sync"
)

var encountersManagerMap *sync.Map
var encounterManagerTimeStep float64

// this file governs the duration of encounters
type EncounterEpoch struct {
	enc               *model.Encounter
	epochEnd          float64 //store only the end of epoch, not the whole one
	lastEncounterTime float64 //this will help to calculate the duration as the end
}

func InitEncManager(defaultTimeStep float64) {
	encounterManagerTimeStep = defaultTimeStep
	encountersManagerMap = &sync.Map{}
}

// this function checks if a new encounter is consecutive
// to an existing one
func CheckConsecutive(enc_c *model.Encounter, epoch_c *Epoch) bool {
	past := GetEncounterEpoch(enc_c)
	if past == nil { //no past encounter - it is by default consecutive to itself
		//so store it after this function
		return true
	}
	//check if the current epoch follows the one in memory
	last_enc_end := past.epochEnd
	return last_enc_end == epoch_c.prev
}

// get the encounter epoch struct
func GetEncounterEpoch(enc *model.Encounter) *EncounterEpoch {
	// get the nodes as keys
	key := [2]int{model.NodeIdInt(enc.Node1), model.NodeIdInt(enc.Node2)}
	// check for existing EncounterEpoch for the key
	var tmp any
	var ok bool
	if tmp, ok = encountersManagerMap.Load(key); !ok {
		return nil
	}
	return tmp.(*EncounterEpoch)
}

// update encounter
// if this this the first time the nodes meet, store the encounter data
// elsewhere, store for the original meeting, the end of current epoch
// and the current encounter time
// so we can calculate it at the end
func UpdateEncounterChain(enc *model.Encounter, epoch *Epoch) {
	// get the nodes as keys
	key := [2]int{model.NodeIdInt(enc.Node1), model.NodeIdInt(enc.Node2)}
	// get the nodes as keys
	past := GetEncounterEpoch(enc)
	if past == nil { //this is the first time the nodes meet
		encountersManagerMap.Store(key, &EncounterEpoch{enc: enc, epochEnd: epoch.now, lastEncounterTime: enc.Time})
	} else {
		//store the new data
		//but remember to store the original encounter with the new epoch
		//so we can "concatenate" the duration based on the original encounter
		//and the new epoch
		//Also, save the time of the current encounter
		//so at the end, we will be able to correctly calculate the duration
		encountersManagerMap.Store(key, &EncounterEpoch{enc: past.enc, epochEnd: epoch.now, lastEncounterTime: enc.Time})
	}
}

// this function finalizes the duration and sends the encounter
// to the channel
func FinalizeDuration(enc *model.Encounter, epoch *Epoch) {
	orig_data := GetEncounterEpoch(enc)
	final_enc := orig_data.enc
	//we saved the last encounter time in the chain
	//so we can now calculate the estimated duration
	final_enc.Duration = float32(math.Max(orig_data.lastEncounterTime-final_enc.Time, encounterManagerTimeStep))
	// and..send it to the channel
	encounterChan <- final_enc
	//now,set the current encounter for next iteration
	//as the past encounter
	key := [2]int{model.NodeIdInt(orig_data.enc.Node1), model.NodeIdInt(orig_data.enc.Node2)}
	encountersManagerMap.Delete(key)
	//now update the new encounter
	UpdateEncounterChain(enc, epoch)
}

// this function is called at the end of the simulator
// to get the last encounters that were left in the manager
func EmptyEncountersManager() {
	// Create a wait group to wait for all iterations to finish
	var wg sync.WaitGroup

	// Start iterating over the map's keys and values
	encountersManagerMap.Range(func(key, value interface{}) bool {
		wg.Add(1)
		go func(k, v interface{}) {
			defer wg.Done()
			tmp_v := v.(*EncounterEpoch)
			final_enc := tmp_v.enc
			//we saved the last encounter time in the chain
			//so we can now calculate the estimated duration
			final_enc.Duration = float32(math.Max(tmp_v.lastEncounterTime-final_enc.Time, encounterManagerTimeStep))
			// and..send it to the channel
			encounterChan <- final_enc

		}(key, value)
		return true
	})

	// Wait for all iterations to finish
	wg.Wait()
	//finished adding the encounters
	log.Info("finished emptying the encounters manager")
}

// this function sends the encounter when it is appropriate
func CheckEncounter(enc *model.Encounter, epoch *Epoch) {
	//if the encounters are
	if CheckConsecutive(enc, epoch) {
		UpdateEncounterChain(enc, epoch)
	} else { //it is time to send the encounter
		FinalizeDuration(enc, epoch)
	}
}
