package datamodel

import "encoding/json"

// a simple three-dimensional coordinate location
type Coord struct {
	X float64
	Y float64
	Z float64
}

func (*Coord) Id() LocationType {
	return LocationTypeCoord
}

func (c *Coord) Marshall() ([]byte, error) {
	d := struct {
		X float64
		Y float64
		Z float64
	}{
		X: c.X,
		Y: c.Y,
		Z: c.Z,
	}
	return json.Marshal(d)
}

func (c *Coord) Unmarshal(b []byte) error {
	d := struct {
		X float64
		Y float64
		Z float64
	}{}
	err := json.Unmarshal(b, &d)
	if err != nil {
		return err
	}
	c.X = d.X
	c.Y = d.Y
	c.Z = d.Z
	return nil
}

func (c *Coord) ConvertToCoord() *Coord {
	return c
}
