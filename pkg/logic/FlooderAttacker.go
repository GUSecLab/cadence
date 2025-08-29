package logic

import (
	model "marathon-sim/datamodel"
	"sync"

	logger "github.com/sirupsen/logrus"
)

type FlooderAttacker struct {
	log *logger.Logger
}

func (m *FlooderAttacker) Init(log *logger.Logger) {
	m.log = log
}

func (m *FlooderAttacker) GetType() AdversaryApproach {
	return Flooder
}

func (m *FlooderAttacker) GetApproachString() string {
	return "Flooder"
}

// document messages from the regular node to the modifier, but do not deliver anything - it is a black hole
// therefore, the DB is updated on the messaging that the dropper receives and that's it
func (m *FlooderAttacker) OneSidedAttack(encounter *model.Encounter, modifier model.NodeId, attackerMap *sync.Map) {
	// iterate thru messages on attacker,
	//create fake ones
	//to flood the net with fake messages
	attackerMap.Range(func(k, v any) bool {
		newMessage_fake := v.(*Message)
		newMessage_fake.RecordHop(modifier, encounter.Time)
		//generate fake id - but does not know the amount of nodes...
		newMessage_fake.MessageId = GenerateRandomId(10000000000)
		//set a fake message bit
		newMessage_fake.FakeMessage = true
		//fake creation time and TTLs
		newMessage_fake.CreationTime = 0
		newMessage_fake.TTLHops = 100
		newMessage_fake.TTLTime = 10000
		//hand over to the general handler,
		//with the attacker as sender
		//and the benign as receiver
		return true
	})

}
