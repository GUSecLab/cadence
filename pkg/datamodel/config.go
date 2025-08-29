/**
 * configuration for a test run, includes default values for all args
 *
 */

package datamodel

import (
	"fmt"
	"crypto/rand"
)

type Config struct {
	TopLevel TopLevelConfig		`json:"top_level"`
	WebServer WebServerConfig	`json:"web_server"`
	Simulation SimulationConfig	`json:"simulation"`
	CLI CLIConfig				`json:"cli"`
}

type TopLevelConfig struct {
	// top level
	Log 		string `json:"log"`
	DataBase 	string `json:"db"`
	DBFile 		string `json:"dbfile"`
	TimeFormat 	string `json:"time_format"`
	Seed		int 	`json:"seed"`
}

type WebServerConfig struct {
	// web server
	Port 		int 	`json:"port"`
	Host 		string	`json:"host"`
}

type SimulationConfig struct {
	// simulation

	// experiment name must be unique for each run
	ExperimentName    	string  `json:"experiment_name"`
	
	StartTime			float32 `json:"start_time"`
	EndTime				float32 `json:"end_time"`
	DatasetName       	string  `json:"dataset_name"`
	ConditionsFile    	string  `json:"conditions_file"`
	Logic			  	string  `json:"logic"`
	LogicFile         	string  `json:"logic_file"`
	MessagesFile      	string  `json:"messages_file"`
	MessagesTemplate   	string  `json:"messages_template"`
	GeneratorScript   	string  `json:"generator_script"`
	GenerationType	 	int     `json:"message_generation_type"`
	MinBufferSize     	int     `json:"min_buffer_size"`
	MaxBufferSize     	int     `json:"max_buffer_size"`
	MinMessageSize    	float32 `json:"min_message_size"`
	MaxMessageSize    	float32 `json:"max_message_size"`
	TimeStep          	float32 `json:"time_step"`
	EncounterIsolator 	float32 `json:"encounter_isolator"`
	NEncountersSplit   	int     `json:"N_encounters_split"`
	KTopNode			int     `json:"K_top_nodes"`
	BufferType 	        string  `json:"buffer_type"`
	Profiling 		  	bool    `json:"profiling"`
}

type CLIConfig struct {
	// CLI command line interface - for lens import
	Lens 		string `json:"lens"`
	Path		string `json:"path"`
	Name		string `json:"name"`
}

/**
initializes the configuration to default values
*/
func MakeDefaultConfig() *Config {

	DefaultConfig := new(Config)

	// generate a new experiment name using UUID - some stuff from stackoverflow
	// https://stackoverflow.com/a/65607935/22193553
	s := make([]byte, 10)
	rand.Read(s)
	DefaultConfig.Simulation.ExperimentName = "TEST-" + fmt.Sprintf("%x", s)[2 : 12]

	DefaultConfig.TopLevel.Log = "DEBUG"
	DefaultConfig.TopLevel.DataBase = "mysql"
	DefaultConfig.TopLevel.DBFile = "marathon:marathon@tcp(127.0.0.1:3306)/marathondev?charset=utf8mb4&parseTime=True&loc=Local"
	DefaultConfig.TopLevel.TimeFormat = "2006-01-02 15:04:05.000"
	DefaultConfig.TopLevel.Seed = 12345
	DefaultConfig.WebServer.Port = 8080
	DefaultConfig.WebServer.Host = "localhost"
	DefaultConfig.Simulation.StartTime = 1200000000
	DefaultConfig.Simulation.EndTime = 1340000000
	DefaultConfig.Simulation.DatasetName = "default config geolife experiment"
	DefaultConfig.Simulation.ConditionsFile = "conditions.json"
	DefaultConfig.Simulation.Logic = "broadcast"
	DefaultConfig.Simulation.LogicFile = "../../pkg/logic/logic_config_geolife.json"
	DefaultConfig.Simulation.MessagesFile = "../../../marathon-files/messages.json"
	DefaultConfig.Simulation.MessagesTemplate = "messages_template.json"
	DefaultConfig.Simulation.GenerationType = 1
	DefaultConfig.Simulation.MinBufferSize = 50
	DefaultConfig.Simulation.MaxBufferSize = 50
	DefaultConfig.Simulation.MinMessageSize = 1.0
	DefaultConfig.Simulation.MaxMessageSize = 1.0
	DefaultConfig.Simulation.TimeStep = 60.0
	DefaultConfig.Simulation.EncounterIsolator = 60.0
	DefaultConfig.Simulation.NEncountersSplit = 2
	DefaultConfig.Simulation.KTopNode = 64
	DefaultConfig.Simulation.BufferType = "simple"
	DefaultConfig.Simulation.Profiling = false
	DefaultConfig.CLI.Lens = "geolife"
	DefaultConfig.CLI.Path = "../../marathon-mobility-data/geolife"
	DefaultConfig.CLI.Name = "geolife"

	return DefaultConfig
}


