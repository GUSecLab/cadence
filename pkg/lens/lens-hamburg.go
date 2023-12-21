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

// Geolife is a Lens for the geolife dataset
type Hamburg struct {
	r       *regexp.Regexp
	log     *logger.Logger
	nodeMap map[string]model.NodeId
}

func (h *Hamburg) Init(log *logger.Logger) {
	h.log = log
	var err error
	h.r, err = regexp.Compile("Data/([0-9]+)/")
	if err != nil {
		panic(err)
	}
	h.nodeMap = make(map[string]model.NodeId)
}

func (h *Hamburg) GetLocationType() model.LocationType {
	return model.LocationTypeLongLat
}

func (h *Hamburg) getNodeId(nodeString string) model.NodeId {
	id, ok := h.nodeMap[nodeString]
	if !ok {
		// not found, so add a new node
		h.nodeMap[nodeString] = model.NodeId(len(h.nodeMap))
		return h.nodeMap[nodeString]
	} else {
		return id
	}
}

func (h *Hamburg) processFile(path, datasetName string) error {
	map_events := make(map[*model.Event]bool)
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
	nodes := make(map[model.NodeId]bool)
	events := make([]model.Event, 0)
	eventCounter := 0
	var nodeId model.NodeId
	r := csv.NewReader(strings.NewReader(buf))
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		latlong := &model.LatLongAlt{}
		if latlong.Lat, err = strconv.ParseFloat(record[2], 64); err != nil {
			return err
		}
		if latlong.Lon, err = strconv.ParseFloat(record[3], 64); err != nil {
			return err
		}
		latlong.Alt = 0.0

		// reconstruct the timestamp
		timestamp := record[4]
		layout := "2006-01-02 15:04:05"
		t, err := time.Parse(layout, timestamp)
		if err != nil {
			// Handle error
			h.log.Warn("error parsing hamburg timestamp")
		}

		marshalledLocation, err := latlong.Marshall()
		if err != nil {
			h.log.Warn("skipping line (can't marshall location)")
			continue
		}
		node := record[1]
		node_id, _ := strconv.Atoi(node)
		nodeId = model.NodeId(node_id)
		nodes[nodeId] = true
		h.log.Debugf("node is %v", nodeId)
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
		ok := map_events[&e]
		if ok {
			continue
		}
		map_events[&e] = true
		events = append(events, e)
		if len(events) > 1000 {
			result := model.DB.Create(&events)
			if result.Error != nil {
				h.log.Warnf("cannot add record: %v", result.Error)
			} else {
				eventCounter += len(events)
				h.log.Debugf("added %v events to DB", eventCounter)
			}
			events = nil
		}
		h.log.Debugf("adding a new event: %v", e)
	}

	// if there are left over events, add it to DB
	if len(events) > 0 {
		result := model.DB.Create(&events)
		if result.Error != nil {
			h.log.Warnf("cannot add record: %v", result.Error)
		} else {
			eventCounter += len(events)
		}
	}
	h.log.Infof("added %v events to DB from '%v'", eventCounter, path)
	for n := range nodes {
		// add nodeID to node_list table
		var nl model.NodeList
		if r := model.DB.Find(&nl, "dataset_name=? and node=?", datasetName, nodeId); r.Error != nil || r.RowsAffected < 1 {
			nl.DatasetName = datasetName
			nl.Node = n
			if r := model.DB.Save(&nl); r.Error != nil {
				h.log.Warnf("cannot add nodeID")
			}
		}
	}

	return nil
}

func (h *Hamburg) Import(path string, datasetName string) error {

	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".csv" {
				h.log.Debugf("processing %v\n", path)
				if err := h.processFile(path, datasetName); err != nil {
					h.log.Infof("error occurred while parsing '%v': %v", path, err)
				}
			}
			return nil
		})
	if err != nil {
		h.log.Warn(err)
	}

	return nil
}
