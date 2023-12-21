package datamodel

import (
	"encoding/json"

	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
)

var geo *ellipsoid.Ellipsoid

// latitude, longitude, and altitude
type LatLongAlt struct {
	Lat float64
	Lon float64
	Alt float64
}

func (*LatLongAlt) Id() LocationType {
	return LocationTypeLongLat
}

func (l *LatLongAlt) Marshall() ([]byte, error) {
	d := struct {
		Lat float64
		Lon float64
		Alt float64
	}{
		Lat: l.Lat,
		Lon: l.Lon,
		Alt: l.Alt,
	}
	return json.Marshal(d)
}

func (l *LatLongAlt) Unmarshal(b []byte) error {

	d := struct {
		Lat float64
		Lon float64
		Alt float64
	}{}
	err := json.Unmarshal(b, &d)
	if err != nil {
		return err
	}
	l.Lat = d.Lat
	l.Lon = d.Lon
	l.Alt = d.Alt
	return nil
}

// Converts the latitutude, longitude, and altitude to a model based on WGS84;
// see https://gisgeography.com/wgs84-world-geodetic-system/.  This allows us to
// compute Euclidean distances between coordinates.
func (l *LatLongAlt) ConvertToCoord() *Coord {
	if geo == nil {
		geo2 := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)
		geo = &geo2
	}
	x, y, z := geo.ToECEF(l.Lat, l.Lon, l.Alt)
	return &Coord{
		X: x,
		Y: y,
		Z: z,
	}
}
