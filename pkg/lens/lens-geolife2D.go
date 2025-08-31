package lens

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	model "marathon-sim/datamodel"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
)

// Geolife is a Lens for the geolife dataset
type Geolife2D struct {
	r       *regexp.Regexp
	log     *logger.Logger
	nodeMap map[string]model.NodeId
}

func (g *Geolife2D) Init(log *logger.Logger) {
	g.log = log
	var err error
	g.r, err = regexp.Compile("Data/([0-9]+)/")
	if err != nil {
		panic(err)
	}
	g.nodeMap = make(map[string]model.NodeId)
}

func (g *Geolife2D) GetLocationType() model.LocationType {
	return model.LocationTypeLongLat2D
}

func (g *Geolife2D) getNodeId(nodeString string) model.NodeId {
	id, ok := g.nodeMap[nodeString]
	if !ok {
		// not found, so add a new node
		g.nodeMap[nodeString] = model.NodeId(len(g.nodeMap))
		return g.nodeMap[nodeString]
	} else {
		return id
	}
}

func (g *Geolife2D) processFile(path, datasetName string) error {
	node := g.r.FindStringSubmatch(path)[1]
	nodeId := g.getNodeId(node)
	g.log.Debugf("node is %v", nodeId)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)
	// skip the first 6 lines
	for i := 0; i < 6; i++ {
		fileScanner.Scan()
	}
	// read the rest into memory
	buf := ""
	for fileScanner.Scan() {
		buf += fileScanner.Text() + "\n"
	}

	events := make([]model.Event, 0)
	eventCounter := 0

	r := csv.NewReader(strings.NewReader(buf))
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
		if latlong.Alt, err = strconv.ParseFloat(record[3], 64); err != nil {
			return err
		}
		// the dataset weirdly uses -777 to indicate invalid altitude, so just
		// assume 0.  Also, use a range so that we don't run into floating point
		// conversion issues
		if math.Abs(-777-latlong.Alt) < 1.0 {
			latlong.Alt = 0
		}

		// reconstruct the timestamp
		datetimeString := record[5] + "T" + record[6] + "Z"
		t, err := time.Parse(time.RFC3339, datetimeString)
		if err != nil {
			g.log.Warnf("cannot parse datetime '%v': %v", datetimeString, err)
			continue
		}

		marshalledLocation, err := latlong.Marshall()
		if err != nil {
			g.log.Warn("skipping line (can't marshall location)")
			continue
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
		events = append(events, e)
		if len(events) > 1000 {
			result := model.DB.Create(&events)
			if result.Error != nil {
				g.log.Warnf("cannot add record: %v", result.Error)
			} else {
				eventCounter += len(events)
				g.log.Debugf("added %v events to DB", eventCounter)
			}
			events = nil
		}
		g.log.Debugf("adding a new event: %v", e)
	}

	// if there are left over events, add it to DB
	if len(events) > 0 {
		result := model.DB.Create(&events)
		if result.Error != nil {
			g.log.Warnf("cannot add record: %v", result.Error)
		} else {
			eventCounter += len(events)
		}
	}
	g.log.Infof("added %v events to DB from '%v'", eventCounter, path)

	// add nodeID to node_list table
	var nl model.NodeList
	if r := model.DB.Find(&nl, "dataset_name=? and node=?", datasetName, nodeId); r.Error != nil || r.RowsAffected < 1 {
		nl.DatasetName = datasetName
		nl.Node = nodeId
		if r := model.DB.Save(&nl); r.Error != nil {
			g.log.Warnf("cannot add nodeID")
		}
	}

	return nil
}

func (geolife *Geolife2D) Import(path string, datasetName string) error {

	err := filepath.Walk(path,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".plt" {
				geolife.log.Debugf("processing %v\n", path)
				if err := geolife.processFile(path, datasetName); err != nil {
					geolife.log.Warnf("error occurred while parsing '%v': %v", path, err)
				}
			}
			return nil
		})
	if err != nil {
		geolife.log.Warn(err)
		return err
	}

	return nil
}
