package datamodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToCoord(t *testing.T) {
	latLongAlt1 := &LatLongAlt{
		Lat: 38.907971,
		Lon: -77.071614,
		Alt: 0.0,
	}
	latLongAlt2 := &LatLongAlt{
		Lat: 38.900680,
		Lon: -77.048783,
		Alt: 0.0,
	}

	coord1 := latLongAlt1.ConvertToCoord()
	coord2 := latLongAlt2.ConvertToCoord()

	// according to https://www.omnicalculator.com/other/latitude-longitude-distance,
	// the distance should be 2135.5m

	var d EuclidDistance
	dist, err := d.Distance(coord1, coord2)
	assert.Nil(t, err, "Error should be nil")

	assert.InDelta(t, 2135.5, dist, 10.0, "Distance should be 2135.5m")
}

func TestUnmarshal(t *testing.T) {
	data := []byte(`{"Lat": 38.907971, "Lon": -77.071614, "Alt": 123.456}`)
	expected := &LatLongAlt{
		Lat: 38.907971,
		Lon: -77.071614,
		Alt: 123.456,
	}

	var l LatLongAlt
	err := l.Unmarshal(data)
	assert.Nil(t, err, "Error should be nil")
	assert.Equal(t, expected, &l, "Unmarshalled value should match expected")
}
