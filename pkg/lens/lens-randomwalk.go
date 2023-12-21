package lens

import (
	"bufio"
	"compress/gzip"
	"fmt"
	model "marathon-sim/datamodel"
	"os"

	logger "github.com/sirupsen/logrus"
)

type RandomWalk struct {
	log *logger.Logger
}

func (walk *RandomWalk) Init(log *logger.Logger) {
	walk.log = log
}

func (walk *RandomWalk) GetLocationType() model.LocationType {
	return model.LocationTypeLongLat
}

func (walk *RandomWalk) Import(filename string, datasetName string) error {
	file, err := os.Open(filename)

	if err != nil {
		walk.log.Fatal(err)
	}

	gz, err := gzip.NewReader(file)
	if err != nil {
		walk.log.Fatal(err)
	}
	defer file.Close()
	defer gz.Close()
	scanner := bufio.NewScanner(gz)

	events := make([]model.Event, 0)

	lineCounter := 0
	eventCounter := 0

	users := make(map[model.NodeId]interface{})

	// iterate thru the gzipped file a line at a time
	for scanner.Scan() {
		lineCounter += 1
		var user int
		var simtime, lat, lon float64
		l := scanner.Text()
		if _, err := fmt.Sscanf(l, "%f,%d,%f,%f", &simtime, &user, &lat, &lon); err != nil {
			walk.log.Warnf("skipping line %v", lineCounter)
			continue
		}

		latlong := &model.LatLongAlt{Lat: lat, Lon: lon}
		marshalledLocation, err := latlong.Marshall()
		if err != nil {
			walk.log.Warnf("skipping line %v (can't marshall location)", lineCounter)
			continue
		}

		users[model.NodeId(user)] = nil

		e := model.Event{
			DatasetName:        datasetName,
			Time:               float64(simtime),
			Node:               model.NodeId(user),
			MarshalledLocation: marshalledLocation,
			X:                  latlong.Lat,
			Y:                  latlong.Lon,
		}
		events = append(events, e)
		if len(events) > 1000 {
			result := model.DB.Create(&events)
			if result.Error != nil {
				walk.log.Warnf("cannot add record: %v", result.Error)
			} else {
				eventCounter += len(events)
				walk.log.Debugf("added %v events to DB", eventCounter)
			}
			events = nil
		}
	}
	if err := scanner.Err(); err != nil {
		walk.log.Warnf("scanner error: %v", err)
	}

	// if there are left over events, add it to DB
	if len(events) > 0 {
		result := model.DB.Create(&events)
		if result.Error != nil {
			walk.log.Warnf("cannot add record: %v", result.Error)
		} else {
			eventCounter += len(events)
		}
	}

	walk.log.Infof("added %v events to DB", eventCounter)

	for u := range users {
		// add nodeID to node_list table
		var nl model.NodeList
		nl.DatasetName = datasetName
		nl.Node = u
		if r := model.DB.Save(&nl); r.Error != nil {
			walk.log.Warnf("cannot add nodeID")

		}
	}

	return nil
}
