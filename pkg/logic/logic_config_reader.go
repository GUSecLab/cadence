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
	ConstantTransferProbability   float32       `json:"randomwalk_transfer_constant"`
	MixedRandomVersion            int           `json:"mixed_randomwalk_version"`
	MixedIdealityVersion          int           `json:"mixed_ideality_version"`
	MaxBufferSize                 int           `json:"max_buffer_size"`
	IdealityCoinFlip              float32       `json:"ideality_coin_flip"`
	IdealityDistanceN             int           `json:"ideality_distance_n"`
	UnderlyningLogic              string        `json:"underlyning_logic"`
	Dataset                       string        `json:"dataset_name"`
	Organizer                     OrganizerType `json:"organizer"`
	RemoveDeliveredMessages       bool          `json:"remove_delivered_messages"`
	DistrictsAmount               int           `json:"districts_amount"`
	P                             float32       `json:"p"`
	Wormhole                      bool          `json:"wormhole"`
	K							  int           `json:"k"` // k-hot vector size for the regions
	KMeansIterations			  int           `json:"kmeans_iterations"`
	MaxN                          int		    `json:"max_n"`
	NFactor		                  int           `json:"n_factor"`
	NoiseSigma                    float64       `json:"noise_sigma"` // sigma for DP noise
	AttackType                    string        `json:"attack_type"` // type of attack to perform for MIRAGE
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
