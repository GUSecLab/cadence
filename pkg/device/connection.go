package device

import (
	model "marathon-sim/datamodel"
	"sync"
	"time"
)

// connections state
// this will store a key of [node1,node2]
// to the struct of information
// regarding the data transfer
var Connections *sync.Map

// a connection struct
// this stucrt includes the init time
// and duration
type Connection struct {
	//init time
	InitTime time.Time
	//duration var
	Duration time.Duration
}

// this function checks if a connection exists between two nodes
func CheckConnectionExists(nodeid1 model.NodeId, nodeid2 model.NodeId) bool {
	//check for existing connections
	//if they exists, abort creation of a new connection

	nodes12 := [2]model.NodeId{nodeid1, nodeid2}
	nodes21 := [2]model.NodeId{nodeid2, nodeid1}
	//check for existing connection 1->2
	if _, ok := Connections.Load(nodes12); ok {
		return true
	}
	//check for existing connection 2->1
	if _, ok := Connections.Load(nodes21); ok {
		return true
	}
	return false
}

// create a connection
func ConnectionCreationOrContinuation(nodeid1 model.NodeId, nodeid2 model.NodeId, init_time time.Time, duration time.Duration) {
	//if there is a connection,
	//do not create a new one
	//just return
	if CheckConnectionExists(nodeid1, nodeid2) {
		return
	}
	C := Connection{InitTime: init_time, Duration: duration}
	//simulate connection setup
	time.Sleep(1 * time.Second)
	//store new connection
	Connections.Store([2]model.NodeId{nodeid1, nodeid2}, C)
}

// randomly taring done connections
// the input is the frequency (in seconds)
// of this probabilistic taredown
// and the probability
func TareDownConnectionsRandomly(f int, p float64) {
	target := Connections
	sleep_time := time.Duration(f) * time.Second
	//inifinte loop that terminates connections
	for {
		// probabilistic taredown
		if !model.TrueWithProbability(p) {
			//nothing happens if probabilisticly,
			//no taredown is needed
		} else {

			// Get a slice of the map keys
			keys := make([]interface{}, 0)
			target.Range(func(k, v interface{}) bool {
				keys = append(keys, k)
				return true
			})

			// Generate a random index
			randIndex := model.Intn(int64(len(keys)))

			// Get the random key-value pair
			randomKV := keys[randIndex]
			value, _ := target.Load(randomKV)
			target.Delete(value)

		}
		//sleep for f seconds
		time.Sleep(sleep_time)
	}

}
