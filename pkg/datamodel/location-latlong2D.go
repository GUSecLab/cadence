package datamodel

import (
	"encoding/json"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/project"
)

// latitude, longitude, and altitude
type LatLongAlt2D struct {
	Lat float64
	Lon float64
	Alt float64
}

func (*LatLongAlt2D) Id() LocationType {
	return LocationTypeLongLat2D
}

func (l *LatLongAlt2D) Marshall() ([]byte, error) {
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

func (l *LatLongAlt2D) Unmarshal(b []byte) error {

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

// Converts the latitutude, longitude to a model based on WGS84;
// see https://github.com/paulmach/orb/tree/master/project
func (l *LatLongAlt2D) ConvertToCoord() *Coord {
	sf := orb.Point{l.Lon, l.Lat}
	merc := project.Point(sf, project.WGS84.ToMercator)

	return &Coord{
		X: merc.X(),
		Y: merc.Y(),
		Z: 0,
	}
}