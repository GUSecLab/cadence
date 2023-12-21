package logic

import (
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type DropperAttacker struct {
	log *logger.Logger
}

func (d *DropperAttacker) Init(log *logger.Logger) {
	d.log = log
}
func (d *DropperAttacker) GetType() AdversaryApproach {
	return Dropper
}
func (d *DropperAttacker) GetApproachString() string {
	return "Dropper"
}

// document messages from the regular node to the dropper, but do not deliver anything - it is a black hole
// therefore, the DB is updated on the messaging that the dropper receives and that's it
func (d *DropperAttacker) OneSidedAttack(encounter *model.Encounter, modifier model.NodeId, attackerMap *sync.Map) {
	//delete all the messages
	//this accomplishes the attacker goal -
	//to create a black hole
	attackerMap = &sync.Map{}
}
