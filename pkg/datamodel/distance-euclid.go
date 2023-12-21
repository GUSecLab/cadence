package datamodel

import (
	"errors"
	"math"
)

type EuclidDistance struct{}

func (EuclidDistance) Name() string {
	return "EuclidDistance"
}

// compute the euclidean distance between two locations
func (EuclidDistance) Distance(l1, l2 Location) (float64, error) {
	if l1.Id() != l2.Id() {
		return -1, errors.New("mismatched types")
	}

	var c1, c2 *Coord
	var x1, y1, z1, x2, y2, z2 float64

	// either directly use the coordinate or if it's a latlong, convert to
	// cartesian first
	switch l1.Id() {
	case LocationTypeCoord:
		c1 = l1.(*Coord)
		c2 = l2.(*Coord)
	case LocationTypeLongLat:
		ll1 := l1.(*LatLongAlt)
		ll2 := l2.(*LatLongAlt)
		c1 = ll1.ConvertToCoord()
		c2 = ll2.ConvertToCoord()
	default:
		return -1, errors.New("unsupported location type")
	}

	x1 = c1.X
	y1 = c1.Y
	z1 = c1.Z
	x2 = c2.X
	y2 = c2.Y
	z2 = c2.Z

	var d float64
	if z1 == 0 && z2 == 0 {
		// let's avoid some math if just a 2D coord
		d = math.Sqrt(math.Pow(x1-x2, 2.0) + math.Pow(y1-y2, 2.0))
	} else {
		d = math.Sqrt(math.Pow(x1-x2, 2.0) + math.Pow(y1-y2, 2.0) + math.Pow(z1-z2, 2.0))
	}

	return d, nil
}
