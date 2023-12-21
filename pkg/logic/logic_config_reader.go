package logic

import (
	"encoding/json"
	"fmt"
	"os"

	logger "github.com/sirupsen/logrus"
)

type LogicConf struct {
	LogicName                     string        `json:"logic"`
	AdversariesFile               string        `json:"adversaries_file"`
	ProfileFile                   string        `json:"profile_file"`
	RandomwalkTransferProbability float32       `json:"randomwalk_transfer_probability"`
	RandomwalkDeleteProbability   float32       `json:"randomwalk_delete_probability"`
	MaxBufferSize                 int           `json:"max_buffer_size"`
	UnderlyningLogic              string        `json:"underlyning_logic"`
	Dataset                       string        `json:"dataset_name"`
	Organizer                     OrganizerType `json:"organizer"`
}

// read configuration file
// and return the config block
func ConfigLogic(conf_file string, log *logger.Logger) *LogicConf {
	file, err := os.Open(conf_file)
	if err != nil {
		fmt.Println("Error opening config file:", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := LogicConf{}
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("Error decoding config file:", err)
	}
	//init the logic engine
	return &config

}
