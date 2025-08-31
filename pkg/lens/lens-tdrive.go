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
	"time"

	logger "github.com/sirupsen/logrus"
)

// TDrive is a Lens for the TDrive dataset
type TDrive struct {
	r       *regexp.Regexp
	log     *logger.Logger
	nodeMap map[string]model.NodeId
}

func (g *TDrive) Init(log *logger.Logger) {
	g.log = log
	var err error
	g.r, err = regexp.Compile("[a-zA-Z0-9]+.txt")
	if err != nil {
		panic(err)
	}
	g.nodeMap = make(map[string]model.NodeId)
}

func (g *TDrive) GetLocationType() model.LocationType {
	return model.LocationTypeLongLat
}

func (c *TDrive) getNodeId(nodeString string) model.NodeId {
	id, ok := c.nodeMap[nodeString]
	if !ok {
		// not found, so add a new node
		c.nodeMap[nodeString] = model.NodeId(len(c.nodeMap))
		return c.nodeMap[nodeString]
	} else {
		return id
	}
}

func (c *TDrive) processFile(path, datasetName string) error {
	nodeId := c.getNodeId(path)
	c.log.Infof("node is %v", nodeId)
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
	seen := make(map[float64]bool)
	events := make([]model.Event, 0)
	eventCounter := 0

	r := csv.NewReader(strings.NewReader(buf))
	r.Comma = ','
	for {
		record, err := r.Read()

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		latlong := &model.LatLongAlt{}
		if latlong.Lon, err = strconv.ParseFloat(record[2], 64); err != nil {
			return err
		}
		if latlong.Lat, err = strconv.ParseFloat(record[3], 64); err != nil {
			return err
		}
		latlong.Alt = 0
		marshalledLocation, err := latlong.Marshall()
		if err != nil {
			c.log.Warn("skipping line (can't marshall location)")
			continue
		}
		// reconstruct the timestamp
		timestamp := record[1]
		layout := "2006-01-02 15:04:05"
		t, err := time.Parse(layout, timestamp)
		if err != nil {
			// Handle error
			c.log.Warn("error parsing tdrive timestamp")
		}
		ecefCoord := latlong.ConvertToCoord()
		e := model.Event{
			DatasetName:        datasetName,
			Time:               float64(t.Unix()),
			Node:               nodeId,
			MarshalledLocation: marshalledLocation,
			X:                  ecefCoord.X,
			Y:                  ecefCoord.Y,
			Z:                  ecefCoord.Z,
		}
		if _, ok := seen[e.Time]; ok {
			continue
		} else {
			seen[e.Time] = true
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

	// add nodeID to node_list table
	var nl model.NodeList
	if r := model.DB.Find(&nl, "dataset_name=? and node=?", datasetName, nodeId); r.Error != nil || r.RowsAffected < 1 {
		nl.DatasetName = datasetName
		nl.Node = nodeId
		if r := model.DB.Save(&nl); r.Error != nil {
			c.log.Warnf("cannot add nodeID")
		}
	}

	return nil
}

func (c *TDrive) Import(path string, datasetName string) error {
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
		return err
	}

	return nil
}
