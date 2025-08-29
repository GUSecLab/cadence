package main

import (
	"encoding/json"
	"fmt"
	model "marathon-sim/datamodel"
	"marathon-sim/lens"
	logics "marathon-sim/logic"
	"os"
	"os/user"
	"runtime/pprof"

	logger "github.com/sirupsen/logrus"
)

// create global log variable
var log *logger.Logger

func main() {

	// generate a config file for cadence
	config := model.MakeDefaultConfig()

	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	// check for valid command line
	// must have at least one parameter specifying run type
	if len(os.Args) < 2 {
		fmt.Println("missing required run type: web, sim, import, list")
		os.Exit(1)
	}

	// check for run type
	runType := os.Args[1]

	if (runType != "web") && (runType != "sim") && (runType != "import") && (runType != "list")&&(runType != "report")&&(runType != "csv") {
		fmt.Println("invalid run type: web, sim, import, list, report, csv")
		os.Exit(1)
	}

	// parse the command line for a json config file

	if len(os.Args) > 2 {
		filename := os.Args[2]
		filedata, err := os.ReadFile(filename)

		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		err = json.Unmarshal(filedata, &config)

		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
	}

	// Start profiling after we parse the config, if necessary 

	if config.Simulation.Profiling {
		log.Info("Profiling has been started.")
		// Run go tool pprof -http=:8080 cadence.prof in the command line to check the result, make sure to install graphviz in your system.
		f, err := os.Create("cadence.prof")
		if err != nil {
			log.Debugf("profiling file is not created: %v", err)
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}	


	// set up the logger
	log = logger.New()
	level, err := logger.ParseLevel(config.TopLevel.Log)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	log.SetLevel(level)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	log.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	log.Info("running with config:\n", config)

	// set up PRNG
	model.Seed(int64(config.TopLevel.Seed))
	log.Infof("random seed is %v", config.TopLevel.Seed)
	log.Info("starting")
	log.Infof("logging at log level %v; all times in UTC", config.TopLevel.Log)
	if path, err := os.Getwd(); err != nil {
		log.Fatalf("cannot get working directory: %v", err)
	} else {
		log.Debugf("running from %v", path)
	}

	// initializing the lenses
	lens.LensInit(log)

	// initialize the database
	model.Init(log, config)

	// initialize the message counter
	logics.InitCounter(log)

	// initiailize the encounter manager - now consant of 20 seconds
	InitEncManager(float64(config.Simulation.TimeStep))

	// running cadence based on specified runType

	switch runType {
	case "import":
		if err := lens.Import(config); err != nil {

			log.Warnf("import raised an error: %v", err)
		}

	case "sim":
		// initialize memory control
		log.Info("defining default memory/buffer")
		logics.InitMemory(log, config)

		// initialize the logic engines in general and the specific engine used
		// logics.LogicEnginesInit(log, config.Simulation.LogicFile)
		logics.LogicEnginesInit(log, config)

		// run simulation logic
		simulate(config, user.Username)

	case "list":
		lens.PrintLenses()

	case "web":
		webService(config)
	
	case "report":
		downloadReport()
	
	case "csv" :
		exportCSV()
	}
	

	// ending the run

	log.Info(" ... ending ... ")

	if config.Simulation.Profiling {
		log.Info("Profiling has been stopped.")
	}

}
