package logic

import (

)

type District struct {
	// id of the district 
	DistrictId int
	
	// centroid of the district 
	X float64
	Y float64
	
	// should be 0!
	Z float64
}