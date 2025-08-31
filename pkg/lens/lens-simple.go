package lens

import (
	"encoding/json"
	model "marathon-sim/datamodel"
	"os"

	logger "github.com/sirupsen/logrus"
)

// SimpleDatasetFormat is a Lens for the SimpleDatasetFormat dataset
// (i.e., datasets created using the gen-simple-dataset tool)
type SimpleDatasetFormat struct {
	log *logger.Logger
}

type simplePosition struct {
	X, Y, Z float64
}

type simpleEvent struct {
	User    int             `json:"user"`
	SimTime int             `json:"simtime"`
	Pos     *simplePosition `json:"position"`
}

func (g *SimpleDatasetFormat) Init(log *logger.Logger) {
	g.log = log
}

func (g *SimpleDatasetFormat) GetLocationType() model.LocationType {
	return model.LocationTypeCoord
}

func (c *SimpleDatasetFormat) processFile(path, datasetName string) error {

	var simpleEvents []simpleEvent

	// read the input file
	inputFilePtr, err := os.Open(path)
	if err != nil {
		c.log.Fatal(err)
	}
	defer inputFilePtr.Close()

	c.log.Info("reading input file")
	err = json.NewDecoder(inputFilePtr).Decode(&simpleEvents)
	if err != nil {
		c.log.Fatal(err)
	}

	events := make([]model.Event, 0, 1000)
	eventCounter := 0

	nodeList := make(map[int]struct{})

	for _, ev := range simpleEvents {

		e := model.Event{
			DatasetName:        datasetName,
			Time:               float64(ev.SimTime),
			Node:               model.NodeId(ev.User),
			MarshalledLocation: nil,
			X:                  ev.Pos.X,
			Y:                  ev.Pos.Y,
			Z:                  ev.Pos.Z,
		}
		events = append(events, e)
		nodeList[ev.User] = struct{}{}

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

	// add nodes to nodeList table
	for nodeId := range nodeList {
		nl := model.NodeList{
			DatasetName: datasetName,
			Node:        model.NodeId(nodeId),
		}
		if r := model.DB.Save(&nl); r.Error != nil {
			c.log.Warnf("cannot add nodeID")
		}

	}

	c.log.Info("Completed import")
	return nil
}

func (c *SimpleDatasetFormat) Import(path string, datasetName string) error {

	if err := c.processFile(path, datasetName); err != nil {
		c.log.Warnf("error occurred while parsing '%v': %v", path, err)
		return err
	}
	return nil
}
