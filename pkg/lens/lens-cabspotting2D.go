package lens

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	model "marathon-sim/datamodel"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	logger "github.com/sirupsen/logrus"
)

// Cabspotting is a Lens for the cabspotting dataset
type Cabspotting2D struct {
	r       *regexp.Regexp
	log     *logger.Logger
	nodeMap map[string]model.NodeId
}

func (g *Cabspotting2D) Init(log *logger.Logger) {
	g.log = log
	var err error
	g.r, err = regexp.Compile("new_[a-zA-Z0-9]+")
	if err != nil {
		panic(err)
	}
	g.nodeMap = make(map[string]model.NodeId)
}

func (g *Cabspotting2D) GetLocationType() model.LocationType {
	return model.LocationTypeLongLat2D
}

func (c *Cabspotting2D) getNodeId(nodeString string) model.NodeId {
	id, ok := c.nodeMap[nodeString]
	if !ok {
		// not found, so add a new node
		c.nodeMap[nodeString] = model.NodeId(len(c.nodeMap))
		return c.nodeMap[nodeString]
	} else {
		return id
	}
}

func (c *Cabspotting2D) processFile(path, datasetName string) error {
	node := c.r.FindStringSubmatch(path)[0]
	nodeId := c.getNodeId(node)
	c.log.Debugf("node is %v", nodeId)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)

	// read the rest into memory
	buf := ""
	for fileScanner.Scan() {
		buf += fileScanner.Text() + "\n"
	}

	events := make([]model.Event, 0)
	eventCounter := 0

	r := csv.NewReader(strings.NewReader(buf))
	r.Comma = ' '
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		latlong := &model.LatLongAlt2D{}
		if latlong.Lat, err = strconv.ParseFloat(record[0], 64); err != nil {
			return err
		}
		if latlong.Lon, err = strconv.ParseFloat(record[1], 64); err != nil {
			return err
		}
		latlong.Alt = 0
		var t float64
		if t, err = strconv.ParseFloat(record[3], 64); err != nil {
			return err
		}
		Coord := latlong.ConvertToCoord()
		marshalledLocation, err := latlong.Marshall()
		if err != nil {
			// this should not happen, but just in case
			c.log.Warnf("failed to marshall latlong: %v, %v", latlong, err)
			continue
		}

		e := model.Event{
			DatasetName:        datasetName,
			Time:               t,
			Node:               nodeId,
			MarshalledLocation: marshalledLocation,
			X:                  Coord.X,
			Y:                  Coord.Y,
			Z:                  Coord.Z,
		}
		
		events = append(events, e)
		if len(events) > 1000 {
			result := model.DB.Create(&events)
			if result.Error != nil {
				c.log.Warnf("cannot add record: %v", result.Error)
			} else {
				eventCounter += len(events)
				c.log.Debugf("added %v events to DB", eventCounter)
			}
			events = nil
		}
		c.log.Debugf("adding a new event: %v", e)
	}

	// if there are left over events, add it to DB
	if len(events) > 0 {
		result := model.DB.Create(&events)
		if result.Error != nil {
			c.log.Warnf("cannot add record: %v", result.Error)
		} else {
			eventCounter += len(events)
		}
	}
	c.log.Infof("added %v events to DB from '%v'", eventCounter, path)

	return nil
}

func (c *Cabspotting2D) Import(path string, datasetName string) error {
	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".txt" {
				c.log.Debugf("processing %v\n", path)
				if err := c.processFile(path, datasetName); err != nil {
					c.log.Warnf("error occurred while parsing '%v': %v", path, err)
				}
			}
			return nil
		})
	if err != nil {
		c.log.Warn(err)
	}

	return nil
}