package logic

import (
	"fmt"
	model "marathon-sim/datamodel"
	"math"
	"sort"
	"sync"

	logger "github.com/sirupsen/logrus"
)

// Point3D represents a 3D point.
type Point3D struct {
	X, Y, Z float32
}

// String returns the string representation of a Point3D.
func (p Point3D) String() string {
	return fmt.Sprintf("Point3D{X: %v, Y: %v, Z: %v}", p.X, p.Y, p.Z)
}

// location popularity struct
type LocPop struct {
	Address Point3D
	Score   int
}

type Cube struct {
	LookupMapGeneral     *sync.Map //the sync.map that stores the small cubes for general profile
	GridSize             float32
	MinPoint             Point3D
	MaxPoint             Point3D
	LookupMapLocation    *sync.Map //the sync.map that stores the small cubes for location profile
	SimilarityMapGeneral *sync.Map //map of similarity
}

// EnclosingCubeMin returns the minimum XYZ point of the enclosing small cube.
func EnclosingCubeMin(minXYZ Point3D, D float32, point Point3D) Point3D {
	xIdx := int(math.Floor(float64((point.X - minXYZ.X) / D)))
	yIdx := int(math.Floor(float64((point.Y - minXYZ.Y) / D)))
	zIdx := int(math.Floor(float64((point.Z - minXYZ.Z) / D)))

	return Point3D{
		X: minXYZ.X + float32(xIdx)*D,
		Y: minXYZ.Y + float32(yIdx)*D,
		Z: minXYZ.Z + float32(zIdx)*D,
	}
}

// general location profile
func GeneralLocationProfileCreation(nodes map[model.NodeId][]model.Event, gridSize float64, epsilon float64, logg *logger.Logger, barrier *sync.WaitGroup) {
	logg.Infof("starting general profile creation")
	//create global counter and global movements map
	GlobalMovementsCount := 0
	for id, geos := range nodes {
		logg.Infof("starting general profile from data of node %v", id)

		// Temporary map to collect updates
		tempUpdates := make(map[Point3D]int)

		for _, geo_event := range geos {
			geo := Point3D{float32(geo_event.X), float32(geo_event.Y), float32(geo_event.Z)}
			// Get the enclosing cube
			Cell := EnclosingCubeMin(Profilier.Area.MinPoint, float32(Profilier.GridData.Distance), geo)

			// Collect updates in the temporary map
			if value, ok := tempUpdates[Cell]; ok {
				tempUpdates[Cell] = value + 1
			} else {
				tempUpdates[Cell] = 1
			}
			GlobalMovementsCount++
		}

		// Apply the collected updates to the sync.Map
		for Cell, increment := range tempUpdates {
			// Load the value using the cell and update it
			Profilier.Area.LookupMapGeneral.Store(Cell, float32(increment)/float32(GlobalMovementsCount))
		}

	}
	//precompute the general similatiry
	SimilarityGeneralPrecompute()
	//update barrier
	barrier.Done()
}

//this function precomputes the general similarity for each address

func SimilarityGeneralPrecompute() {

	CellsDivide := func(key, value interface{}) bool {
		pam := 0.0
		score := 0.0
		addressCube, ok := key.(Point3D) // get the address
		if !ok {
			return true
		}
		// getting the value of pam
		pam = float64(value.(float32))

		InnerCellsDivide := func(key, value interface{}) bool {
			innerAddressCube, ok := key.(Point3D) // get the address
			if !ok {                              // something is wrong
				return true
			}
			if innerAddressCube == addressCube { // not computing the same address
				return true
			}

			if v, ok := value.(float32); ok {
				score += float64(v) / Distance3DPoint(addressCube, innerAddressCube, Profilier.GridData.Gridsize)

			}

			return true // continue iteration
		}

		// Iterating and updating each value
		Profilier.Area.LookupMapGeneral.Range(InnerCellsDivide)

		// Remember the similarity of this address
		Profilier.Area.SimilarityMapGeneral.Store(addressCube, pam+score)
		return true // continue iteration
	}

	// Iterating and updating each value
	Profilier.Area.LookupMapGeneral.Range(CellsDivide) //finished precomputing the general profile
	logg.Info("fininshed creating the general profile")
	// Profilier.Area.LookupMapGeneral = nil //erase the data from here, it is not necessary anymore
}

// create location profile, dump it to a dedicated channel at the end
func LocationProfileCreation(nodes map[model.NodeId][]model.Event, gridSize float64, epsilon float64, logg *logger.Logger) *sync.Map {
	logg.Infof("starting location profile creation")
	//create a leftover map for the actual data for the experiment
	times := sync.Map{}

	for key := range nodes {
		// Get the length of the slice
		originalLen := len(nodes[key])

		// Calculate the index up to which we want to keep the elements
		endIdx := int(math.Max(0.1*float64(originalLen), 1.0))
		//sort the slice according to time, the last element
		sortedSlice := nodes[key]
		sort.Slice(sortedSlice, func(i, j int) bool {
			return sortedSlice[i].Time < sortedSlice[j].Time
		})
		// Use slicing to create a new slice with only the first 30% of the original elements
		newSlice := sortedSlice[:endIdx]
		//save the other data for the experiment
		left_items := sortedSlice[endIdx:]
		if len(left_items) == 0 || len(newSlice) == 0 { //node with no actual data
			continue
		}

		// change - we return the event and not the time here!
		//save the first time of each node
		// times.Store(key, left_items[0].Time)
		times.Store(key, left_items) //store the first event of the node
		// Replace the value of the current key with the new slice
		nodes[key] = newSlice
	}
	//create the barrier for the specific locations creation
	var loc_barrier sync.WaitGroup

	for id, geos_tmp := range nodes {
		loc_barrier.Add(1)
		geos := make([]Point3D, 0)
		for _, geo := range geos_tmp {
			geos = append(geos, Point3D{float32(geo.X), float32(geo.Y), float32(geo.Z)})
		}

		//create the profile location for a specific nodeid
		go SpecificLocationProfileCreation(id, geos, gridSize, epsilon, logg, &loc_barrier)
	}
	loc_barrier.Wait()

	//return the map of times
	return &times

}

// geolocation creation for a specific node
func SpecificLocationProfileCreation(id model.NodeId, geos []Point3D, gridSize float64, epsilon float64, logg *logger.Logger, barrier *sync.WaitGroup) {
	logg.Infof("starting location profile of node %v", id)

	for _, geo := range geos {
		//update the location grid with the geolocation
		UpdateLocationProfile(geo, id)
		//update the amount of update counted,globally/specifically
		tmp_location_value, _ := Profilier.LocationUpdates.LoadOrStore(id, 0)
		Profilier.LocationUpdates.Store(id, tmp_location_value.(int)+1)

	}
	//signal the barrier that this location was created
	barrier.Done()

}

// update location profile
func UpdateLocationProfile(geo Point3D, node model.NodeId) {
	locationProfileScore := 0
	//get the address
	address := EnclosingCubeMin(Profilier.Area.MinPoint, float32(Profilier.GridData.Distance), geo)
	//get the data from the map of the node
	node_data, exists := Profilier.Area.LookupMapLocation.Load(node)
	if !exists {
		//generate the node location
		//and and first visit in the address
		tmp_map := sync.Map{}
		tmp_map.Store(address, 1)
		locationProfileScore = 1
		Profilier.Area.LookupMapLocation.Store(node, &tmp_map)

	} else {
		//get the node data for a specific address
		cell_data, exists := node_data.(*sync.Map).Load(address)

		//get the value of the nodeid per cell
		if exists { //node exists in the cell
			locationProfileScore = cell_data.(int) + 1
		} else { //first time this node is relevant
			locationProfileScore = 1
		}
		node_data.(*sync.Map).Store(address, locationProfileScore)

	}
	//now, update the most popular place for the node, based on this update
	node_data, exists = Profilier.Popularity.Load(node)
	if !exists { //first time for the node
		pCur := &LocPop{address, locationProfileScore}
		Profilier.Popularity.Store(node, pCur)
	} else {
		popScore := node_data.(*LocPop).Score
		if popScore < locationProfileScore { //the current address is more popular
			pCur := &LocPop{address, locationProfileScore}
			Profilier.Popularity.Store(node, pCur)
		}
	}
}

// // calculates the location profile
// func LocationCalculation(nodeid model.NodeId, Address Point3D) float64 {
// 	//eliminate invalid values
// 	tmp_location_value, _ := Profilier.LocationUpdates.LoadOrStore(nodeid, 0)
// 	//this means that the node practically does not exist
// 	if tmp_location_value == 0 {
// 		logg.Infof("Node %v does not exist. Logic problem", model.NodeIdString(nodeid))
// 		return 0
// 	}

// 	//location profile calculation
// 	//get the places of this node
// 	nodedata, ok := Profilier.Area.LookupMapLocation.Load(nodeid)
// 	if !ok {
// 		logg.Infof("error in finding cube: %v", ok)
// 		return 0
// 	}
// 	//find per cell in the cube, the
// 	//profile of the node
// 	node_data, ok := cell.(SmallCube).PersonalProfiles.Load(nodeid)
// 	if !ok {
// 		logg.Infof("error in finding location profile of nodeid %v: %v", model.NodeIdString(nodeid), ok)
// 		return 0
// 	}

// 	score := node_data.(float32)
// 	return float64(score) / float64(tmp_location_value.(int))
// }

// get the most popular cube for a node
// func GetPupularCube(nodeid model.NodeId) Point3D {
// 	var mostPopPoint Point3D
// 	maxVal := 0
// 	// Iterate over the sync.Map
// 	Profilier.Area.LookupMapLocation.Range(func(k, v interface{}) bool {
// 		//get the current address of cube
// 		addressCube, ok := k.(Point3D)
// 		if !ok {
// 			fmt.Println("problem with getting addresses for location profile similarity", addressCube)
// 		}
// 		profiles := v.(SmallCube).PersonalProfiles
// 		nodeValue, ok := profiles.Load(nodeid)
// 		if !ok {
// 			fmt.Print("node is not found in address", model.NodeIdString(nodeid), addressCube)
// 		}
// 		//update the most popular point3D if it is more than
// 		//previous one
// 		if nodeValue.(int) > maxVal {
// 			maxVal = nodeValue.(int)
// 			mostPopPoint = addressCube
// 		}
// 		return true
// 	})
// 	return mostPopPoint
// }