package main

import (
	"encoding/json"
	"io/ioutil"
	model "marathon-sim/datamodel"
	"marathon-sim/lens"
	logics "marathon-sim/logic"
	"os"
	"testing"

	logger "github.com/sirupsen/logrus"
)

// simple general config data example
var GeneralConfigData = `
{
    "top_level": {
	"log": "INFO",
	"db": "mysql",
	"dbfile": "marathon:marathon@tcp(127.0.0.1:3306)/marathon?charset=utf8mb4&parseTime=True&loc=Local",
	"time_format": "2006-01-02 15:04:05.000" ,
	"seed": 12345
  },
  "web_server": {
	"port": 8080,
	"host": "localhost"
  },
  "simulation": {
	"start_time": 1408000000,
	"end_time": 1408500000,
	"dataset_name": "napa",
	"experiment_name": "broadcast_test20",
	"conditions_file": "conditions.json",
	"logic": "broadcast", 
	"logic_file": "../../pkg/logic/logic_config.json",
	"messages_file": "../../../marathon-files/messages.json",
	"messages_template": "messages_template.json",
	"generator_script": "message_generator.py",
	"message_generation_type": 1,
	"random_taredowns": 0,
	"taredowns_freq": 2,
	"taredowns_prob": 0.01,
	"min_buffer_size": 0,
	"max_buffer_size": 0,
	"min_connection_duration": 1,
	"max_connection_duration": 2,
	"min_message_size": 1.0,
	"max_message_size": 1.0
  },
  "cli":{
	"lens": "napa",
	"path": "../../marathon-mobility-data/napa",
	"name": "napa"
  }
}
`

// simple logic config data for logic engine config block
var LogicConfBlock = `{
    "logic": "randomwalk-v1",
    "dataset_name": "napa",
    "adversaries_file": "attackers.json",
    "underlyning_logic": "broadcast",
    "profile_file": "profile_defs.json",
    "randomwalk_transfer_probability": 0.7,
    "randomwalk_delete_probability": 1,
    "randomwalk_transfer_constant": 100,
    "mixed_randomwalk_version": 1,
    "mixed_flood_version": 0,
    "mixed_ideality_version": 0,
    "max_buffer_size": 0,
    "ideality_coin_flip": 0.7,
    "ideality_distance_n": 1
} `

var ConditionJson = `{
	"conditions" : [
	  {
		"name" : "distance200m",
		"type" : "distance",
		"params" : {
		  "dist" : 200.0
		}
	  },
	  {
		"name" : "probability50",
		"type" : "probability",
		"params" : {
		  "prob" : 1.0
		}
	  }
	]
  }`

// configuarion parsing tests

/*
func TestGenerateCliCommand(t *testing.T) {

	// Unmarshal the JSON into a Config struct
	var config Config
	err := json.Unmarshal([]byte(GeneralConfigData), &config)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	//run the generate command and comapre
	confType := "cli"
	command := GenerateCommand(config, confType)
	expected := "./cadence cli import --lens napa -p ../../marathon-mobility-data/napa -n napa --dbtype mysql -D marathon:marathon@tcp(127.0.0.1:3306)/marathon?charset=utf8mb4&parseTime=True&loc=Local"

	if command != expected {
		t.Errorf("The cli commands are not equal. Expected: %s, got: %s", expected, command)
	}
}


func TestGenerateSimCommand(t *testing.T) {

	// Unmarshal the JSON into a Config struct
	var config Config
	err := json.Unmarshal([]byte(GeneralConfigData), &config)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
	//run the generate command and comapre
	confType := "sim"
	command := GenerateCommand(config, confType)
	expected := "./cadence sim --name napa --experiment broadcast_test20 --start 1408000000.000000 --end 1408499968.000000 --seed 12345 --logic broadcast -c conditions.json --mes ../../../marathon-files/messages.json --mesgen 1 --mestemplate messages_template.json --messcript message_generator.py -p ../../pkg/logic/logic_config.json -r 0 -f 2 -b 0.010000 -k 0 -g 0 --maxmes 1.000000 --minmes 1.000000 --maxd 2.000000 --mind 1.000000 --dbtype mysql -D marathon:marathon@tcp(127.0.0.1:3306)/marathon?charset=utf8mb4&parseTime=True&loc=Local"
	if command != expected {
		t.Errorf("The sim commands are not equal. Expected: %s, got: %s", expected, command)
	}
}

func TestGenerateWebCommand(t *testing.T) {

	// Unmarshal the JSON into a Config struct
	var config Config
	err := json.Unmarshal([]byte(GeneralConfigData), &config)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
	//run the generate command and comapre
	confType := "web"
	command := GenerateCommand(config, confType)
	expected := "./cadence web --dbtype mysql -D marathon:marathon@tcp(127.0.0.1:3306)/marathon?charset=utf8mb4&parseTime=True&loc=Local"
	if command != expected {
		t.Errorf("The sim commands are not equal. Expected: %s, got: %s", expected, command)
	}
}
*/

// test the lens
func TestLensInitGHCI(t *testing.T) {
	log = logger.New()
	l, err := logger.ParseLevel("INFO")
	if err != nil {
		t.Error("error in log creation")
		os.Exit(1)
	}
	log.SetLevel(l)
	lens.LensInit(log)
	lens_amount := len(lens.LensStore)
	expected := 15
	if lens_amount != expected {
		t.Errorf("The amount of lens doe not match. Expected: %d, got: %d", expected, lens_amount)
	}

}

// test the db init
func TestDBInit(t *testing.T) {
	//set logger
	log = logger.New()
	l, err := logger.ParseLevel("INFO")
	if err != nil {
		t.Error("error in log creation")
		os.Exit(1)
	}
	log.SetLevel(l)
	lens.LensInit(log)
	// Unmarshal the JSON into a Config struct
	var config *model.Config
	err = json.Unmarshal([]byte(GeneralConfigData), &config)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
	//init the db
	model.Init(log, config)
	// a list of tables
	tablesToMigrate := map[interface{}]string{
		&model.Dataset{}:            "dataset",
		&model.Experiment{}:         "experiment",
		&model.Event{}:              "event",
		&model.Encounter{}:          "encounter",
		&model.EncounteredNodes{}:   "encounter_nodes",
		&model.NodeList{}:           "node_lists",
		&model.MessageDB{}:          "message_db",
		&model.DeliveredMessageDB{}: "delievered_message_db",
		&model.BufferMax{}:          "buffer_max",
		&model.ResultsDB{}:          "results_db",
	}
	//check existance of tables
	for table, name := range tablesToMigrate {
		hasTable := model.DB.Migrator().HasTable(&table)
		if hasTable != true {
			t.Errorf("Didn't create table %v.", name)
		}
	}

}

// check logic engines
func TestLogicEnginesInitCommandGHCI(t *testing.T) {


	
	// Unmarshal the JSON into a Config struct
	var logicConfig logics.LogicConf
	err := json.Unmarshal([]byte(LogicConfBlock), &logicConfig)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}
	// Marshal the JSON object into bytes
	jsonBytes, err := json.Marshal(logicConfig)
	if err != nil {
		log.Info("Error marshaling JSON:", err)
		return
	}

	// Write the JSON bytes to a file
	err = ioutil.WriteFile("tmp_conf.json", jsonBytes, 0644)
	if err != nil {
		log.Info("Error writing file:", err)
		return
	}
	//should fail, just running the init for the logicstore
	var config *model.Config
	err = json.Unmarshal([]byte(GeneralConfigData), &config)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	// this is necessary to prepare the organizer
	logics.InitMemory(log, config)

	// logics.LogicEnginesInit(log, "tmp_conf.json")
	logics.LogicEnginesInit(log, config)
	//get the current logic name from the logic store
	logicEng, err := logics.GetLogicByName(logicConfig.LogicName)
	if err != nil {
		log.Infof("problem with init logic engine %v, abort! %v", logicConfig.LogicName, err)
		return
	}
	// Delete the file
	err = os.Remove("tmp_conf.json")
	if err != nil {
		log.Info("Error deleting file:", err)
		return
	}
	//init it with the block
	expected := logicConfig.LogicName
	result := logicEng.GetLogicName()
	if result != expected {
		t.Errorf("The logic config block has problem creating the logic. Expected: %s, got: %s", expected, result)
	}
}

// test proper creation of conditions
func TestConditionsGHCI(t *testing.T) {
	// Unmarshal the JSON into conditions
	var m map[string][]map[string]any
	var err error
	if err = json.Unmarshal([]byte(ConditionJson), &m); err != nil {
		log.Info(err)
	}
	conditionsMap, ok := m["conditions"]

	if !ok {
		log.Info(err)
	}
	for _, conditionMap := range conditionsMap {
		var areParams bool
		nameVal, ok := conditionMap["name"].(string)
		if !ok {
			log.Info(err, "'name' not properly defined for condition in JSON file")
		}
		typeVal, ok := conditionMap["type"].(string)
		if !ok {
			log.Info(err, "'type' not properly defined for condition in JSON file")
		}
		params, areParams := conditionMap["params"].(map[string]any)

		switch typeVal {

		case "distance":
			if !areParams {
				log.Info(err, "'params' not defined for condition named "+nameVal)
			}
			_, ok := params["dist"].(float64)
			if !ok {
				log.Info(err, "'dist' not defined for condition named "+nameVal)
			}

		case "probability":
			if !areParams {
				log.Info(err, "'params' not defined for condition named "+nameVal)
			}
			_, ok := params["prob"].(float64)
			if !ok {
				log.Info(err, "'prob' not defined for condition named "+nameVal)
			}

		}
	}
}

// test data preparation, a.k.a getting the data from the DB for specific use
// func TestPrep(t *testing.T) {
// 	//set logger
// 	log = logger.New()
// 	l, err := logger.ParseLevel("INFO")
// 	if err != nil {
// 		t.Error("error in log creation")
// 		os.Exit(1)
// 	}
// 	log.SetLevel(l)
// 	lens.LensInit(log)
// 	// Unmarshal the JSON into a Config struct
// 	var config Config
// 	err = json.Unmarshal([]byte(GeneralConfigData), &config)
// 	if err != nil {
// 		t.Errorf("Failed to unmarshal JSON: %v", err)
// 	}
// 	//init the db
// 	model.Init(log, config.TopLevel.DB, config.TopLevel.DB_File)
// 	//get the nodes list
// 	nodes_list, err := model.DB.Table("node_lists").Select("node").Where("dataset_name=?", config.TopLevel.DB).Rows()
// 	if err != nil {
// 		log.Infof("error in fetching the events for profiling")
// 	}

// 	rows, err := model.DB.Table("events").Where("dataset_name=? and time>=? and time<=?", config.Simulation.DatasetName, config.Simulation.StartTime, config.Simulation.EndTime).Order("time").Rows()
// 	if err != nil {
// 		log.Infof("error in fetching the events for profiling")
// 	}

// 	// Unmarshal the JSON into a Config struct
// 	var configLog logics.LogicConf
// 	err = json.Unmarshal([]byte(LogicConfBlock), &configLog)
// 	if err != nil {
// 		t.Errorf("Failed to unmarshal JSON: %v", err)
// 	}
// 	// Marshal the JSON object into bytes
// 	jsonBytes, err := json.Marshal(configLog)
// 	if err != nil {
// 		log.Info("Error marshaling JSON:", err)
// 		return
// 	}

// 	// Write the JSON bytes to a file
// 	err = ioutil.WriteFile("tmp_conf.json", jsonBytes, 0644)
// 	if err != nil {
// 		log.Info("Error writing file:", err)
// 		return
// 	}
// 	// should fail, just running the init for the logicstore
// 	logics.LogicEnginesInit(log, "tmp_conf.json")

// 	// get the current logic name from the logic store
// 	logicEng, err := logics.GetLogicByName(configLog.LogicName)
// 	if err != nil {
// 		log.Infof("problem with init logic engine %v, abort! %v", configLog.LogicName, err)
// 	}
// 	// Delete the file
// 	err = os.Remove("tmp_conf.json")
// 	if err != nil {
// 		log.Info("Error deleting file:", err)
// 	}
// 	//prepare the data
// 	times_data, nodes := logicEng.DataPreparation(rows, nodes_list, log)
// 	//check if it is created
// 	if len(nodes) < 1 {
// 		t.Errorf("The nodes list is problematic.")
// 	}
// 	isEmpty := true
// 	times_data.Range(func(_, _ interface{}) bool {
// 		isEmpty = false // At least one element found, so it's not empty
// 		return false    // Stop the iteration after the first element
// 	})
// 	if isEmpty {
// 		t.Errorf("No times data was created.")
// 	}
// }
