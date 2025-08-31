package datamodel

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

// this is the equivalent of a ConditionType enum
type ConditionType int64

const (
	ConditionTypeUnknown ConditionType = iota
	ConditionTypeNull
	ConditionTypeDistanceInMeters
	ConditionTypeProbailisticMessageSend
	ConditionWhenHaveISawUCondition
	ConditionDoYouHaveEnoughUpdates
)

// an EncounterCondition is a predicate over  an encounter between two nodes
// (i.e., two Events belonging to two nodes).
type EncounterCondition interface {
	GetName() string

	GetType() ConditionType

	// for recording in the DB
	String() string

	IsConditionMet(event1, event2 *Event, node1mes *sync.Map, node2mes *sync.Map, timestamp float64) bool
	//update if needed,given an encounter was established
	UpdateEncounterConsequences(encounter *Encounter)
}

type EncounterConditions []EncounterCondition

type EncounterConditionJSON struct {
	Conditions []map[string]string
}

// reads a JSON-based description file and returns the parsed
// EncounterConditions (i.e., what constitutes an encounter).  This function
// also prepares a string version of the conditions, for inclusion in the
// Experiment table.
func ReadConditionsFile(filename string) (EncounterConditions, string, error) {
	// load conditions
	dat, err := os.ReadFile(filename)
	if err != nil {
		return nil, "", err
	}

	// the format of the JSON file is a map from "conditions" to an array
	// of maps from strings to something
	var m map[string][]map[string]any
	if err = json.Unmarshal(dat, &m); err != nil {
		log.Warn(err)
		return nil, "", err
	}
	conditionsMap, ok := m["conditions"]

	if !ok {
		log.Warn(err)
		return nil, "", err
	}

	conditions := make(EncounterConditions, 0, len(conditionsMap))
	var conditionStrings []string
	for _, conditionMap := range conditionsMap {
		var areParams bool
		nameVal, ok := conditionMap["name"].(string)
		if !ok {
			return nil, "", errors.New("'name' not properly defined for condition in JSON file")
		}
		typeVal, ok := conditionMap["type"].(string)
		if !ok {
			return nil, "", errors.New("'type' not properly defined for condition in JSON file")
		}

		params, areParams := conditionMap["params"].(map[string]any)

		var c EncounterCondition
		switch typeVal {

		case "distance":
			if !areParams {
				return nil, "", errors.New("'params' not defined for condition named " + nameVal)
			}
			dist, ok := params["dist"].(float64)
			if !ok {
				return nil, "", errors.New("'dist' not defined for condition named " + nameVal)
			}
			c = &DistanceCondition{
				name: nameVal,
				d:    dist,
			}
		case "probability":
			if !areParams {
				return nil, "", errors.New("'params' not defined for condition named " + nameVal)
			}
			prob, ok := params["prob"].(float64)
			if !ok {
				return nil, "", errors.New("'prob' not defined for condition named " + nameVal)
			}
			c = &ProbabilityCondition{
				name: nameVal,
				p:    prob,
			}
		//condition for encounter controlled by last time
		//the nodes met
		case "whenhaveisawu":
			if !areParams {
				return nil, "", errors.New("'params' not defined for condition named " + nameVal)
			}
			transtiont, ok := params["transition_time"].(float64)
			if !ok {
				return nil, "", errors.New("'transition_time' not defined for condition named " + nameVal)
			}
			c = &WhenHaveISawUCondition{
				name:               nameVal,
				minimal_transition: transtiont,
				encounters_memory:  &sync.Map{},
			}

			//condition for encounter controlled by last time
			//the nodes met
		case "doYouHaveEnoughUpdates":
			c = &DoYouHaveEnoughUpdates{
				name: nameVal,
			}
		}
		conditions = append(conditions, c)
		conditionStrings = append(conditionStrings, c.String())
	}

	return conditions, strings.Join(conditionStrings, " and "), nil
}
