package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

type Config struct {
	TopLevel   TopLevelConfig   `json:"top_level"`
	Server     ServerConfig     `json:"web_server"`
	Simulation SimulationConfig `json:"simulation"`
	Cli        CliConfig        `json:"cli"`
}

type TopLevelConfig struct {
	LogType    string `json:"log"`
	DB         string `json:"db"`
	DB_File    string `json:"dbfile"`
	TimeFormat string `json:"time_format"`
	Seed       int    `json:"seed"`
}

type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}
type SimulationConfig struct {
	StartTime         float32 `json:"start_time"`
	EndTime           float32 `json:"end_time"`
	DatasetName       string  `json:"dataset_name"`
	ExperimentName    string  `json:"experiment_name"`
	ConditionFile     string  `json:"conditions_file"`
	LogicName         string  `json:"logic"`
	MessageFile       string  `json:"messages_file"`
	LogicFile         string  `json:"logic_file"`
	MinBufferSize     int     `json:"min_buffer_size"`
	MaxBufferSize     int     `json:"max_buffer_size"`
	TimeStep          float32 `json:"time_step"`
	EncounterIsolator float32 `json:"encounter_isolator"`
	MinMessageSize    float32 `json:"min_message_size"`
	MaxMessageSize    float32 `json:"max_message_size"`
	MessageGeneration int     `json:"message_generation_type"`
	MessageTemplate   string  `json:"messages_template"`
	GeneratorScript   string  `json:"generator_script"`
	NSplit            int     `json:"N_encounters_split"`
}

type CliConfig struct {
	LensType string `json:"lens"`
	Path     string `json:"path"`
	Name     string `json:"name"`
}

// EnumerateStructFields generates a help message that enumerates the fields of a struct.
func EnumerateStructFields(structName string, v interface{}) string {
	t := reflect.TypeOf(v)
	if t.Kind() != reflect.Struct {
		return ""
	}

	message := fmt.Sprintf("Fields of %s:\n", structName)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name
		fieldTag := field.Tag.Get("json")
		message += fmt.Sprintf("%s (%s)\n", fieldName, fieldTag)
	}

	return message
}

func HelpFunc() {
	config := Config{}
	configHelp := EnumerateStructFields("Config", config)
	fmt.Println(configHelp)

	serverConfig := ServerConfig{}
	serverConfigHelp := EnumerateStructFields("ServerConfig", serverConfig)
	fmt.Println(serverConfigHelp)

	simulationConfig := SimulationConfig{}
	simulationConfigHelp := EnumerateStructFields("SimulationConfig", simulationConfig)
	fmt.Println(simulationConfigHelp)

	cliConfig := CliConfig{}
	cliConfigHelp := EnumerateStructFields("CliConfig", cliConfig)
	fmt.Println(cliConfigHelp)
}

// read configuration file
func GetConfiguration(conf_file string, conf_type []string) string {
	file, err := os.Open(conf_file)
	if err != nil {
		fmt.Println("Error opening config file:", err)
		return "error"
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := Config{}
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("Error decoding config file:", err)
		return "error"
	}
	//generate the command to run
	return GenerateCommand(config, conf_type)
}

// function that generates the input string for the simulator
func GenerateCommand(conf Config, conf_type []string) string {
	return_string := ""
	switch conf_type[0] {
	case "sim":
		{
			return_string = "./cadence sim --name " + conf.Simulation.DatasetName + " --experiment " + conf.Simulation.ExperimentName +
				" --start " + fmt.Sprintf("%f", conf.Simulation.StartTime) + " --end " + fmt.Sprintf("%f", conf.Simulation.EndTime) + " --seed " + fmt.Sprintf("%d", conf.TopLevel.Seed) +
				" --logic " + conf.Simulation.LogicName +
				" -c " + conf.Simulation.ConditionFile +
				" --mes " + conf.Simulation.MessageFile +
				" --mesgen " + fmt.Sprintf("%d", conf.Simulation.MessageGeneration) +
				" --mestemplate " + conf.Simulation.MessageTemplate +
				" --messcript " + conf.Simulation.GeneratorScript +
				" --nsplits " + fmt.Sprintf("%d", conf.Simulation.NSplit) +
				" -p " + conf.Simulation.LogicFile +
				" -k " + fmt.Sprintf("%d", conf.Simulation.MaxBufferSize) +
				" -g " + fmt.Sprintf("%d", conf.Simulation.MinBufferSize) +
				" -z " + fmt.Sprintf("%f", conf.Simulation.TimeStep) +
				" --maxmes " + fmt.Sprintf("%f", conf.Simulation.MaxMessageSize) +
				" --minmes " + fmt.Sprintf("%f", conf.Simulation.MinMessageSize) +
				" --dbtype " + conf.TopLevel.DB +
				" -D " + conf.TopLevel.DB_File
		}

	case "cli":
		switch conf_type[1] { //switch for the cli commands
		case "lenses":
			{
				return_string = "./cadence cli lenses " +
					" --dbtype " + conf.TopLevel.DB +
					" -D " + conf.TopLevel.DB_File
			}

		default:
			{
				return_string = "./cadence cli import --lens " + conf.Cli.LensType + " -p " + conf.Cli.Path + " -n " + conf.Cli.Name +
					" --dbtype " + conf.TopLevel.DB +
					" -D " + conf.TopLevel.DB_File
			}
		}
	case "web":
		{
			return_string = "./cadence web " + "--dbtype " + conf.TopLevel.DB +
				" -D " + conf.TopLevel.DB_File + " -p " + fmt.Sprintf("%d", conf.Server.Port)
		}

	case "help":
		{
			HelpFunc()
			return_string = ""
		}
	}

	return return_string
}
