/*

tracks the amount of times we want to attempt to 
pass a message 

each node has its own N value Map 

*/

package logic

import (
	"sync"
)

type NValueMap struct {
	Map *sync.Map
}

func NewNValueMap() *NValueMap {
	return &NValueMap{
		Map: &sync.Map{},
	}
}

type MessageDistrictPair struct {
	MessageId string 
	CurrentDistrictId int
}

// returns the value of N
// if the value does not exist, return 0 and false
func (n *NValueMap) Get(messageId string, currentDistrictId int) (int, bool) {
	value, ok := n.Map.Load(MessageDistrictPair{MessageId: messageId, CurrentDistrictId: currentDistrictId})
	if !ok {
		return 1, false
	}
	return value.(int), true
}

// sets the value of N
func (n *NValueMap) Set(messageId string, currentDistrictId int, value int) {
	n.Map.Store(MessageDistrictPair{MessageId: messageId, CurrentDistrictId: currentDistrictId}, value)
}

