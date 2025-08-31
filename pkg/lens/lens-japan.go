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

// Japan is a lens for the YJMOB100K human mobility dataset

type Japan struct {
	r       *regexp.Regexp // this is not used
	log     *logger.Logger
	nodeMap map[string]model.NodeId
}

func (j *Japan) Init(log *logger.Logger) {
	j.log = log
	var err error
	j.r, err = regexp.Compile("Data/([0-9]+)/")
	if err != nil {
		panic(err)
	}
	j.nodeMap = make(map[string]model.NodeId)
}

func (j *Japan) GetLocationType() model.LocationType {
	return model.LocationTypeCoord
}

// returns a nodeID (an int) for the node, if the node does not exist
// in the map, it is added 
func (j *Japan) getNodeId(nodeString string) model.NodeId {
	id, ok := j.nodeMap[nodeString]
	if !ok {
		// not found, so add a new node
		j.nodeMap[nodeString] = model.NodeId(len(j.nodeMap)) // self incrementing
		return j.nodeMap[nodeString]
	} else {
		return id
	}
}

// adds data from the buffered reader to the database
func (j *Japan) addToDatabase(r *csv.Reader, path string, datasetName string) error {

	nodes := make(map[model.NodeId]bool) // create a new node map to keep track of the nodes
	events := make([]model.Event, 0) // create an empty set of events
	eventCounter := 0
	var nodeId model.NodeId

	j.log.Debugf("loading data from reader to database")

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// logic for parsing the actual location specified by the record

		coord := &model.Coord{}
		x, y := 0.0, 0.0
		if x, err = strconv.ParseFloat(record[3], 64); err != nil {
			return err
		}
		if y, err = strconv.ParseFloat(record[4], 64); err != nil {
			return err
		}

		// convert to proper scale in meters

		coord.X = x * 500.0
		coord.Y = y * 500.0
		coord.Z = 0.0

		// reconstruct the time from the record

		// extract time from record
		var days, timeslot float64
		if days, err = strconv.ParseFloat(record[1], 64); err != nil {
			return err
		}
		if timeslot, err = strconv.ParseFloat(record[2], 64); err != nil {
			return err
		}

		// since the actual starting date is not specified, we 
		// will use 2024-01-01 as an arbitrary starting date

		var baseTime time.Time
		if baseTime, err = time.Parse("2006-01-02", "2024-01-01"); err != nil {
			return err
		}

		// use data to calculate the time since the base date
		totalMinutes := days * 24 * 60 + timeslot * 30

		time := baseTime.Add(time.Minute * time.Duration(totalMinutes))

		marshalledLocation, err := coord.Marshall()
		if err != nil {
			j.log.Warn("skipping line (can't marshall location)")
			continue
		}

		node, _ := strconv.Atoi(record[0])
		nodeId = model.NodeId(node)
		nodes[nodeId] = true
		j.log.Debugf("node is %v", nodeId)

		e := model.Event{
			DatasetName:        datasetName,
			Time:               float64(time.Unix()),
			Node:               nodeId,
			MarshalledLocation: marshalledLocation,
			X:                  coord.X,
			Y:                  coord.Y,
			Z:                  0.0,
		}

		events = append(events, e)
		if len(events) > 1000 {
			result := model.DB.Create(&events)
			if result.Error != nil {
				j.log.Warnf("cannot add record: %v", result.Error)
			} else {
				eventCounter += len(events)
				j.log.Debugf("added %v events to DB", eventCounter)
			}
			events = nil
		}
		j.log.Debugf("adding a new event: %v", e)

	}

	// if there are left over events, add it to DB
	if len(events) > 0 {
		result := model.DB.Create(&events)
		if result.Error != nil {
			j.log.Warnf("cannot add record: %v", result.Error)
		} else {
			eventCounter += len(events)
		}
	}
	j.log.Infof("added %v events to DB from '%v'", eventCounter, path)
	for n := range nodes {
		// add nodeID to node_list table
		var nl model.NodeList
		if r := model.DB.Find(&nl, "dataset_name=? and node=?", datasetName, nodeId); r.Error != nil || r.RowsAffected < 1 {
			nl.DatasetName = datasetName
			nl.Node = n
			if r := model.DB.Save(&nl); r.Error != nil {
				j.log.Warnf("cannot add nodeID")
			}
		}
	}

	return nil
}

// processFile reads the file at path, and adds the events to the database
func (j *Japan) processFile(path, datasetName string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	j.log.Debug("file opened successfully, initiating scanner")

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	// read the rest into memory
	buf := ""
	line := 0
	maxBufferSize := 10000 // arbitrary

	fileScanner.Scan() // skip the first line, it contains headings

	for fileScanner.Scan() {
		line++
		buf += fileScanner.Text() + "\n"

		// every 10000 lines, log progress, add to database, clear buffer
		if (line % maxBufferSize == 0) {

			j.log.Debugf("read %v lines", line)
			r := csv.NewReader(strings.NewReader(buf))
		
			j.log.Debug("adding events to database")
			err = j.addToDatabase(r, path, datasetName)
			if err != nil {
				return err
			}
			j.log.Debug("clearing buffer")
			buf = ""
		}
	}

	// adding leftover events in buffer to db

	if buf != "" {
		r := csv.NewReader(strings.NewReader(buf))
		
		j.log.Debug("adding remainder events to database")
		err = j.addToDatabase(r, path, datasetName)
		if err != nil {
			return err
		}
		j.log.Debug("clearing buffer")
		buf = ""
	}
	return nil
}

func (j *Japan) Import(path string, datasetName string) error {

	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".csv" {
				j.log.Debugf("processing %v\n", path)

				
				if err := j.processFile(path, datasetName); err != nil {
					j.log.Infof("error occurred while parsing '%v': %v", path, err)
				}
				
			}
			return nil
		})
	if err != nil {
		j.log.Warn(err)
		return err
	}

	return nil
}
