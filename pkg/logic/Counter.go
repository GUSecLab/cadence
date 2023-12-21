package logic

import (
	"sync"

	logger "github.com/sirupsen/logrus"
)

// regarding the data transfer
var Counter *MessageCounter

// the memory structure
type MessageCounter struct {
	MessagesQueue *sync.Map
	log           *logger.Logger
	mutex         *sync.Mutex
}

// init the counter
func InitCounter(logg *logger.Logger) {
	Counter = new(MessageCounter)
	Counter.MessagesQueue = &sync.Map{}
	Counter.log = logg
	Counter.mutex = new(sync.Mutex)
}

// change the counter for a specific message
func ChangeMessageCount(messageID string, to_add int) {
	Counter.mutex.Lock()
	defer Counter.mutex.Unlock()

	// Retrieve the current count for the messageID
	countInterface, ok := Counter.MessagesQueue.Load(messageID)
	if !ok {
		// If the messageID is not found, initialize the count to 1
		Counter.MessagesQueue.Store(messageID, to_add)
	} else {
		// If the messageID is found, increment the count by 1
		count := countInterface.(int)
		Counter.MessagesQueue.Store(messageID, count+to_add)
	}
}

// get the average amount of copies in the system
func GetAvgCopies() float64 {
	total_amount := 0
	num_messages := 0
	Counter.MessagesQueue.Range(func(k, v any) bool {
		//the number of copies are added to the total number
		//and the amount of messages is incremented
		total_amount += v.(int)
		num_messages++
		return true
	})
	return float64(total_amount) / float64(num_messages)
}

// get the currect load based on number of copies
func GetLoadandAvg() (float64, float64) {
	total_amount := 0
	num_messages := 0
	Counter.MessagesQueue.Range(func(k, v any) bool {
		//the number of copies are added to the total number
		//and the amount of messages is incremented
		total_amount += v.(int)
		num_messages++
		return true
	})
	avg := 0.0
	if num_messages != 0 {
		avg = float64(total_amount) / float64(num_messages)
		// Process the result
	} else {
		// Handle the division by zero error
		avg = 0
	}
	return float64(total_amount), avg
}
