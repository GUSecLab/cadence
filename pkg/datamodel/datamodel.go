// This package defines the data model for Cadence.
// It uses gorm (https://gorm.io/) an ORM model for Golang.
package datamodel

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"strconv"
	"sync"

	logger "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite" // Sqlite driver based on GGO

	"gorm.io/gorm"
)

// a reference to the DB GORM object
var DB *gorm.DB

// our logger
var log *logger.Logger

// a node identifier (basically, an int)
type NodeId int

// return a string of the nodeid
func NodeIdString(i NodeId) string {
	return strconv.Itoa(int(i))
}

// return an int of the nodeid
func NodeIdInt(i NodeId) int {
	return int(i)
}

// A `Dataset` describes very high-level information an imported human movement
// dataset.  The actual (time,location) tuples are stored as `Event`s
type Dataset struct {
	DatasetName    string `gorm:"primaryKey"`
	DateImported   sql.NullTime
	CoordType      LocationType
	CompleteImport bool
}

type NodeList struct {
	DatasetName string `gorm:"primaryKey"`
	Node        NodeId `gorm:"primaryKey"`
}

// struct for stroing the buffer maximum usage value
type BufferMax struct {
	ExperimentName string `gorm:"primaryKey"`
	DatasetName    string `gorm:"primaryKey"`
	Node           NodeId `gorm:"primaryKey"`
	Max            float32
}

// An `Event“ is a node occuring at a place and time.  The `X`, `Y`, and `Z`
// parameters are the location of the event.  Note that if the underlying
// location is lat,long,alt, then `X`, `Y`, and `Z` are calculated by projecting
// the equivalent location onto the globe.
type Event struct {
	DatasetName          string  `gorm:"primaryKey;index:dstime;index:nodetime;index:nodetime2,priority:1"`
	Time                 float64 `gorm:"primaryKey;index:dstime;index:nodetime;index:nodetime2,priority:2"`
	Node                 NodeId  `gorm:"primaryKey;index:dstime;index:nodetime;index:nodetime2,priority:3"`
	MarshalledLocation   []byte
	X                    float64 // the coordinate, in cartesian form
	Y                    float64
	Z                    float64
	UnmarshalledLocation Location `gorm:"-"` // can't store a interface{} in the DB
}

// This defines the Experiment table in the DB.  In a nutshell, it describes a
// particular experiment, where an experiment is a simulation with a particular
// configuration run on a dataset.
type Experiment struct {
	ExperimentName     string `gorm:"primaryKey"`
	DatasetName        string
	Investigator       string
	DateStarted        sql.NullTime
	DateFinished       sql.NullTime
	DistanceConditions string
	CommandLine        string
}

// a message struct, for DB purposes
type MessageDB struct {
	//experiment name
	ExperimentName string
	// a unique, unchanging message ID
	MessageId string

	// official sender
	Sender int

	//the two nodes that are involved
	Sender_Node   int
	Reciever_Node int
	// message type
	Type string

	// The intended official destination.  If it's Unicast, then this should be castable
	// to a NodeId. If it's multicast, then to a list of NodeIds.
	Destination string

	Payload string

	// the time the message was originated
	CreationTime float64

	// the time the message was transfered
	TransferTime float64
	//the path of this message in a string
	Path string
	//popularity flags - negative means it is not
	//popular in this means. The MPop is for the
	//most popular, the Lpop for the least.
	MPop int `gorm:"type:tinyint"`
	LPop int `gorm:"type:tinyint"`

	//fake message boolean - default is false
	FakeMessage bool

	//ttl values
	TTLHops int
	TTLSecs int
	//hops passed
	Hops int
	//size of the message
	Size float32
}

// a message struct, for DB purposes
type DeliveredMessageDB struct {
	//experiment name
	ExperimentName string
	// a unique, unchanging message ID
	MessageId string

	// official sender
	Sender int
	// The intended official destination.  If it's Unicast, then this should be castable
	// to a NodeId. If it's multicast, then to a list of NodeIds.
	Destination string

	Payload string
	// the time the message was transfered
	DeliverTime float64
	//the path of this message in a string
	Path string
	//fake message boolean - default is false
	FakeMessage bool

	//ttl values
	TTLHops int `json:"ttl_hops"`
	TTLSecs int `json:"ttl_secs"`

	//hops passed
	Hops int `json:"hops_passed"`
	//size of the message
	Size float32
}

// an `Encounter` is two nodes in close proximity
type Encounter struct {
	DatasetName    string  `gorm:"primaryKey,priority:5;index:expname,priority:2;index:n1,priority:2;index:n2,priority:2"`
	Distance       float64 `gorm:"primaryKey,priority:6;"`
	ExperimentName string  `gorm:"primaryKey,priority:1;index:expname,priority:1;index:n1,priority:1;index:n2,priority:1"`
	Time           float64 `gorm:"primaryKey,priority:2"`
	Node1          NodeId  `gorm:"primaryKey,priority:3;index:n1,priority:3"`
	Node2          NodeId  `gorm:"primaryKey,priority:4;index:n2,priority:3"`
	Duration       float32 //the duration of the encounter
	X              float32
	Y              float32
	Z              float32
}

// an `Encounter` is two nodes in close proximity
type ExperimentFamily struct {
	//family description
	FamilyDataset  string  // the dataset
	FamilyDistance float64 //distance to be equal in conditions
	//member info
	Logic          string
	ExperimentName string `gorm:"primaryKey"`
}

func (e ExperimentFamily) String() string {
	return e.Logic + "_" + e.ExperimentName
}

// an `Encounter` is two nodes in close proximity
type DatasetEncounterEnumeration struct {
	DatasetName string  `gorm:"primaryKey,priority:1;index:expname,priority:1;index:n1,priority:1;index:n2,priority:1"`
	Distance    float32 `gorm:"primaryKey,priority:2;index:expname,priority:2;index:n1,priority:2;index:n2,priority:2"`
	Duration    float32 `gorm:"primaryKey,priority:3;index:expname,priority:3;index:n1,priority:3;index:n2,priority:3"`
	Complete    bool    //saving the message bandwidth
}

// this DB table just lists the nodes that were encountered
type EncounteredNodes struct {
	ExperimentName string `gorm:"primaryKey"`
	Node           NodeId `gorm:"primaryKey"`
	Count          int
	FirstEncounter float64
}

// this DB table just lists the results of an experiment
type ResultsDB struct {
	ExperimentName    string  `gorm:"primaryKey"`
	LatSec            float32 //latency in seconds (average)
	LatHop            float32 //latency in hops (average)
	MaxBuf            float32 //max buffer
	MaxBand           int     //max bandwidth
	NetLoad           int     //network load
	Throughput        float32 //ratio of delivered messages
	NumMessages       int     //number of messages in total
	AvgCopiesMessages float64 //average amount of copies of messages
	PeakLoad          float64 //the pick load during the experiment
	AverageLoad       float64 //average load during the experiment
}

// this DB table store the bandwidth of the encounters
type Bandwidths struct {
	Dataset        string  `gorm:"primaryKey,priority:1"`
	Distance       float64 `gorm:"primaryKey,priority:2"`
	ExperimentName string  `gorm:"primaryKey,priority:3"`
	Logic          string  `gorm:"primaryKey,priority:4"`
	Hash           uint64
	Bandwidth      float32
}

// this DB table just lists the results of an experiment
type EpochLoad struct {
	ExperimentName string  `gorm:"primaryKey,priority:4"`
	Now            float64 `gorm:"primaryKey,priority:1"`
	Prev           float64 `gorm:"primaryKey,priority:2"`
	Load           float64 `gorm:"primaryKey,priority:3"`
	AvgLoad        float64
}

// Calculate the hash of the timestamp and nodes
func CalculateHash(t float64, n1 int, n2 int) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(fmt.Sprintf("%.6f%d%d", t, n1, n2)))
	return hash.Sum64()
}

// Record encounters.  This function shoudl be started as a goroutine.  It waits
// for incoming encounters and records them in the database, in batches for
// efficiency.
func RecordEncounters(experimentName string, encounterChan chan *Encounter, barrier *sync.WaitGroup) {
	const batchsize = 1024 // an arbitrary choice
	encounters := make([]*Encounter, 0, batchsize)

	// this is essentially a set
	allEncounteredNodes := make(map[NodeId]*EncounteredNodes)

	for encounter := range encounterChan {

		// add this encounter to our list of encounters
		encounters = append(encounters, encounter)
		// update our counters
		if _, ok := allEncounteredNodes[encounter.Node1]; !ok {
			allEncounteredNodes[encounter.Node1] = &EncounteredNodes{
				ExperimentName: experimentName,
				Node:           encounter.Node1,
				Count:          1,
				FirstEncounter: encounter.Time,
			}
		} else {
			allEncounteredNodes[encounter.Node1].Count += 1
		}
		if _, ok := allEncounteredNodes[encounter.Node2]; !ok {
			allEncounteredNodes[encounter.Node2] = &EncounteredNodes{
				ExperimentName: experimentName,
				Node:           encounter.Node2,
				Count:          1,
				FirstEncounter: encounter.Time,
			}
		} else {
			allEncounteredNodes[encounter.Node2].Count += 1
		}

		// if we've reached our batch size, send them to the DB
		if len(encounters) >= batchsize {
			if r := DB.Create(&encounters); r.Error != nil {
				log.Warnf("failed to record encounters: %v", r.Error)
			}
			encounters = nil // reset the buffer
		}
	}

	// if we get here, that means that the encounterChan has been closed.

	// do we have any left over?
	if len(encounters) > 0 {
		// if we have any left over in the queue, flush them to the DB
		if r := DB.Create(&encounters); r.Error != nil {
			log.Warnf("failed to record encounters: %v", r.Error)
		}
	}

	// finally, dump the encounters to the encountered_nodes table
	for _, e := range allEncounteredNodes {
		if r := DB.Save(&e); r.Error != nil {
			log.Warn("failed to record node in encounter list: ", r.Error)
		}
	}

	barrier.Done()
}

// Record epoch loads.  This function should be started as a goroutine.  It waits
// for incoming epochloads and records them in the database, in batches for
// efficiency.
func RecordEpochLoad(experimentName string, epochChan chan *EpochLoad, barrier *sync.WaitGroup) {
	const batchsize = 1000 // an arbitrary choice
	epochLoads := make([]*EpochLoad, 0, batchsize)

	for epl := range epochChan {

		// add this epochload to our list of epochloads
		epochLoads = append(epochLoads, epl)

		// if we've reached our batch size, send them to the DB
		if len(epochLoads) >= batchsize {
			if r := DB.Create(&epochLoads); r.Error != nil {
				log.Warnf("failed to record epoch loads: %v", r.Error)
			}
			epochLoads = nil // reset the buffer
		}
	}

	// if we get here, that means that the channel has been closed.

	// do we have any left over?
	if len(epochLoads) > 0 {
		// if we have any left over in the queue, flush them to the DB
		if r := DB.Create(&epochLoads); r.Error != nil {
			log.Warnf("failed to record epoch loads: %v", r.Error)
		}
	}
	barrier.Done()
}

// Record epoch loads.  This function should be started as a goroutine.  It waits
// for incoming epochloads and records them in the database, in batches for
// efficiency.
func RecordBandwidth(experimentName string, bandChan chan *Bandwidths, barrier *sync.WaitGroup) {
	const batchsize = 1000 // an arbitrary choice
	Bandwidth := make([]*Bandwidths, 0, batchsize)

	for ba := range bandChan {

		// add this epochload to our list of Bandwidth
		Bandwidth = append(Bandwidth, ba)

		// if we've reached our batch size, send them to the DB
		if len(Bandwidth) >= batchsize {
			if r := DB.Create(&Bandwidth); r.Error != nil {
				log.Warnf("failed to record epoch loads: %v", r.Error)
			}
			Bandwidth = nil // reset the buffer
		}
	}

	// if we get here, that means that the channel has been closed.

	// do we have any left over?
	if len(Bandwidth) > 0 {
		// if we have any left over in the queue, flush them to the DB
		if r := DB.Create(&Bandwidth); r.Error != nil {
			log.Warnf("failed to record Bandwidths: %v", r.Error)
		}
	}
	barrier.Done()
}

// returns true iff the dataset has already been imported
func IsImported(datasetName string) (bool, error) {
	var e Event
	r := DB.Take(&e, "dataset_name=?", datasetName)
	if r.Error != nil || r.RowsAffected != 1 {
		return false, r.Error
	} else {
		return true, nil
	}
}

// retrieves all of the datasets and experiments from the database
func GetDatasets() ([]string, error) {
	var datasets []Dataset
	if r := DB.Find(&datasets, "complete_import=true"); r.Error != nil {
		return nil, r.Error
	}
	ds_array := make([]string, 0, len(datasets))
	for _, d := range datasets {
		ds_array = append(ds_array, d.DatasetName)
	}
	return ds_array, nil
}

// retrieves all of the messages from the database
func GetMessagesDB() ([]MessageDB, error) {
	var messages []MessageDB
	if r := DB.Find(&messages, "complete_import=true"); r.Error != nil {
		return nil, r.Error
	}

	return messages, nil
}

// retrieves all of the datasets and experiments from the database
func GetDatasetsAndExperiments() ([]Experiment, error) {
	var experiments []Experiment
	r := DB.Find(&experiments)
	if r.Error != nil {
		return nil, r.Error
	} else {
		return experiments, nil
	}
}

// retrieves all of the datasets and experiments from the database
func GetExperimentsFamily() ([]ExperimentFamily, error) {
	var families []ExperimentFamily
	r := DB.Find(&families)
	if r.Error != nil {
		return nil, r.Error
	} else {
		return families, nil
	}
}

// initializes the data model, creating (and updating!) tables if necessary
func Init(mainLogger *logger.Logger, dbType, dbFileOrDSN string) {
	var err error

	log = mainLogger

	switch dbType {
	case "sqllite":
		DB, err = gorm.Open(sqlite.Open(dbFileOrDSN), &gorm.Config{})
	case "mysql":
		DB, err = gorm.Open(mysql.Open(dbFileOrDSN), &gorm.Config{})
	default:
		log.Fatalf("invalid or unsupported database type: %v", dbType)
	}

	if err != nil {
		log.Fatal(err)
	} else {
		log.Infof("using database '%v'", dbFileOrDSN)
	}

	// a list of blank structs
	tablesToMigrate := []interface{}{
		&Dataset{},
		&Experiment{},
		&Event{},
		&Encounter{},
		&EncounteredNodes{},
		&NodeList{},
		&MessageDB{},
		&DeliveredMessageDB{},
		&BufferMax{},
		&ResultsDB{},
		&EpochLoad{},
		&DatasetEncounterEnumeration{},
		&Bandwidths{},
		&ExperimentFamily{},
	}
	// use GORM to create a DB table for each of the above structs
	for _, table := range tablesToMigrate {
		if err = DB.AutoMigrate(table); err != nil {
			log.Fatal(err)
		}
	}
}

func (m *Encounter) Copy() *Encounter {
	newEnc := &Encounter{
		DatasetName:    m.DatasetName,
		Distance:       m.Distance,
		ExperimentName: m.ExperimentName,
		Time:           m.Time,
		Node1:          m.Node1,
		Node2:          m.Node2,
		Duration:       m.Duration,
		X:              m.X,
		Y:              m.Y,
		Z:              m.Z,
	}

	return newEnc
}

// copy a resultsdb struct
func (r *ResultsDB) Copy() *ResultsDB {
	newR := &ResultsDB{
		ExperimentName:    r.ExperimentName,
		LatSec:            r.LatSec,
		LatHop:            r.LatHop,
		MaxBuf:            r.MaxBuf,
		MaxBand:           r.MaxBand,
		NetLoad:           r.NetLoad,
		Throughput:        r.Throughput,
		NumMessages:       r.NumMessages,
		AvgCopiesMessages: r.AvgCopiesMessages,
		PeakLoad:          r.PeakLoad,
		AverageLoad:       r.AverageLoad,
	}

	return newR
}
