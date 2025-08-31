/*

NOTE: addressing logic requires additional setup steps
that the main simulation logic currently does not support,
so addressing tests only tests simple logic initialization

*/

package main

import (
	"encoding/json"
	"fmt"
	model "marathon-sim/datamodel"
	"marathon-sim/lens"
	logics "marathon-sim/logic"
	"os"
	"testing"

	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var addressingTestConfig *model.Config

var addressingTestEncounter12 *model.Encounter
var addressingTestEncounter13 *model.Encounter

var addressingTestLogger *logger.Logger

func SetupaddressingLogicTest() {

	addressingTestConfig = model.MakeDefaultConfig()

	filename := "configs/unit_tests/config_addressing_test.json"
	filedata, err := os.ReadFile(filename)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err = json.Unmarshal(filedata, &addressingTestConfig); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	addressingTestLogger = logger.New()
	level, err := logger.ParseLevel(addressingTestConfig.TopLevel.Log)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	addressingTestLogger.SetLevel(level)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	addressingTestLogger.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	addressingTestLogger.Info("running with config:\n", addressingTestConfig)

	// set up PRNG
	model.Seed(int64(addressingTestConfig.TopLevel.Seed))
	addressingTestLogger.Infof("random seed is %v", addressingTestConfig.TopLevel.Seed)
	addressingTestLogger.Info("starting")
	addressingTestLogger.Infof("logging at log level %v; all times in UTC", addressingTestConfig.TopLevel.Log)
	if path, err := os.Getwd(); err != nil {
		addressingTestLogger.Fatalf("cannot get working directory: %v", err)
	} else {
		addressingTestLogger.Debugf("running from %v", path)
	}

	// initializing the lenses
	lens.LensInit(addressingTestLogger)

	// initialize the database
	model.Init(addressingTestLogger, addressingTestConfig)

	// initialize the message counter
	logics.InitCounter(addressingTestLogger)

	// initiailize the encounter manager - now consant of 20 seconds
	InitEncManager(float64(addressingTestConfig.Simulation.TimeStep))

	logics.InitMemory(addressingTestLogger, addressingTestConfig)
	logics.LogicEnginesInit(addressingTestLogger, addressingTestConfig)

	// create a test encounter between two nodes

	addressingTestEncounter12 = &model.Encounter{
		DatasetName:    "test dataset",
		Distance:       25.0,
		ExperimentName: "test experiment",
		Time:           1200000000,
		Node1:          1,
		Node2:          2,
		Duration:       10.0,
		X:              0.0,
		Y:              0.0,
		Z:              0.0,
	}

	addressingTestEncounter13 = &model.Encounter{
		DatasetName:    "test dataset",
		Distance:       25.0,
		ExperimentName: "test experiment",
		Time:           1200000000,
		Node1:          1,
		Node2:          3,
		Duration:       10.0,
		X:              100.0,
		Y:              100.0,
		Z:              0.0,
	}

	// initialize node memory for our three test nodes

	logics.InitNodeMemory(addressingTestConfig, addressingTestEncounter12.Node1)
	logics.InitNodeMemory(addressingTestConfig, addressingTestEncounter12.Node2)
	logics.InitNodeMemory(addressingTestConfig, addressingTestEncounter13.Node2)
	// kick off helper function for messages channel
	logics.InitChan()

}

// test if config parameters are being properly set
// using json file
func TestAddressingInitLogic(t *testing.T) {

	// setup the config and the block

	addressingTestConfig = model.MakeDefaultConfig()

	filename := "configs/unit_tests/config_addressing_test.json"
	filedata, err := os.ReadFile(filename)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	if err = json.Unmarshal(filedata, &addressingTestConfig); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	logicFile := addressingTestConfig.Simulation.LogicFile

	// setup the logger

	addressingTestLogger = logger.New()
	level, err := logger.ParseLevel(addressingTestConfig.TopLevel.Log)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	addressingTestLogger.SetLevel(level)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	addressingTestLogger.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	// create the block
	block := logics.ConfigLogic(logicFile, addressingTestLogger)

	// check that block is not null

	assert.NotNil(t, block)

	// check that the block has correct information
	assert.Equal(t, "addressing", block.LogicName)
}
