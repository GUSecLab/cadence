package logic

import (
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// this is the equivalent of a ConditionType enum
type AdversaryApproach int

const (
	Dropper AdversaryApproach = iota
	Flooder
)

type Approach interface {
	//init
	Init(log *logger.Logger)
	//get the type of the attacker
	GetType() AdversaryApproach
	//get the type of the attacker as a string
	GetApproachString() string
	//manipulate the attacker queue
	OneSidedAttack(encounter *model.Encounter, modifier model.NodeId, attackerMap *sync.Map)
}
