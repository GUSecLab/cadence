package datamodel

import "errors"

// this is the equivalent of a LocationType enum
type LocationType int64

const (
	LocationTypeUnknown LocationType = iota
	LocationTypeCoord
	LocationTypeLongLat
)

// a `Location` is an interface that marshals and unmarshals locations.
// Crucially, it must implement a ConvertToCoord() function that converts the
// location to an (X,Y,Z) coordinate.
type Location interface {
	// a unique identifier for this type of Location
	Id() LocationType

	// converter to JSON
	Marshall() ([]byte, error)

	// converter from JSON
	Unmarshal([]byte) error

	// convert to a Coord (i.e., an X,Y,Z coordinate) using the WGS84 coordinate
	// system; see https://gisgeography.com/wgs84-world-geodetic-system/
	ConvertToCoord() *Coord
}

func Unmarshall(bytes []byte, t LocationType) (Location, error) {
	switch t {
	case LocationTypeCoord:
		var c *Coord = new(Coord)
		if err := c.Unmarshal(bytes); err != nil {
			return nil, err
		} else {
			return c, nil
		}
	case LocationTypeLongLat:
		var l *LatLongAlt = new(LatLongAlt)
		if err := l.Unmarshal(bytes); err != nil {
			return nil, err
		} else {
			return l, nil
		}
	case LocationTypeUnknown:
		return nil, errors.New("invalid location type")
	default:
		return nil, errors.New("invalid location type")
	}
}
