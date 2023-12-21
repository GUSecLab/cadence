package main

import (
	"fmt"
	model "marathon-sim/datamodel"
	"marathon-sim/lens"
	logics "marathon-sim/logic"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/akamensky/argparse"
	logger "github.com/sirupsen/logrus"
)

// let's make life easier by making the logger a global variable
var log *logger.Logger

func main() {
	// let's figure out who is running this thing
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	parser := argparse.NewParser("cadence", "the Cadence simulator")

	// top level args
	logLevel := parser.Selector("d", "level", []string{"INFO", "DEBUG", "WARN"},
		&argparse.Options{
			Help:    "logging level",
			Default: "INFO",
		})
	// seed arg
	seed := parser.Int("s", "seed", &argparse.Options{
		Help:    "random number seed",
		Default: int(time.Now().UnixNano()),
	})
	// database type
	dbType := parser.Selector("t", "dbtype", []string{"sqllite", "mysql"},
		&argparse.Options{
			Help:    "database type",
			Default: "sqllite",
		})
	// database file
	dbFile := parser.String("D", "dbfile", &argparse.Options{
		Help:    "path to database file or data source name (DSN)",
		Default: "gorm.db",
	})

	// web server args
	commandWeb := parser.NewCommand("web", "run web server")
	webPort := commandWeb.Int("p", "port", &argparse.Options{
		Help:    "port to listen on",
		Default: 8080,
	})
	webHost := commandWeb.String("i", "host", &argparse.Options{
		Help:    "host to listen on",
		Default: "localhost",
	})

	// simulation args
	commandSim := parser.NewCommand("sim", "run a simulation")
	// timing args
	simTimeStart := commandSim.Float("S", "start", &argparse.Options{
		Help:    "start time (in UNIX time)",
		Default: 0.0,
	})
	simTimeEnd := commandSim.Float("E", "end", &argparse.Options{
		Help:    "end time (in UNIX time); defaults to end of run",
		Default: -1.0,
	})
	// dataset name
	simDataset := commandSim.String("n", "name", &argparse.Options{
		Help:     "dataset name",
		Required: true,
	})
	// experiment name
	simExp := commandSim.String("e", "experiment", &argparse.Options{
		Help:     "experiment name",
		Required: true,
	})
	// researcher name

	simInvestigator := commandSim.String("i", "investigator", &argparse.Options{
		Help:    "investigator's name",
		Default: user.Username,
	})
	// general conditions file

	simConditions := commandSim.String("c", "conditions", &argparse.Options{
		Help:     "path to conditions JSON file",
		Required: true,
	})
	// logic engine name
	simLogic := commandSim.String("L", "logic", &argparse.Options{
		Help:     "logic engine to use (" + strings.Join(logics.GetInstalledLogicEngines(), ",") + ")",
		Required: true,
	})
	// engine logic conf file
	simLogicFile := commandSim.String("p", "logicpf", &argparse.Options{
		Help:     "defines the logic file to use",
		Required: true,
	})
	// messages files to use
	simMessages := commandSim.String("m", "mes", &argparse.Options{
		Help:     "message output file to use",
		Required: true,
	})
	simMessagesGen := commandSim.String("a", "mesgen", &argparse.Options{
		Help:     "message generation type to use",
		Required: true,
	})
	simMessagesTemp := commandSim.String("j", "mestemplate", &argparse.Options{
		Help:     "message generation type to use",
		Required: true,
	})
	simGeneratorScript := commandSim.String("q", "messcript", &argparse.Options{
		Help:     "message generation type to use",
		Required: true,
	})

	//buffer defs
	simMaxBuffer := commandSim.Int("k", "maxb", &argparse.Options{
		Help:     "defines the maximum size of worker buffer",
		Required: true,
	})
	simMinBuffer := commandSim.Int("g", "minb", &argparse.Options{
		Help:     "defines the minimum size of worker buffer",
		Required: true,
	})
	//messages confs
	simMinMesSize := commandSim.Float("v", "minmes", &argparse.Options{
		Help:     "defines the minimum size of a message",
		Required: true,
	})
	simMaxMesSize := commandSim.Float("w", "maxmes", &argparse.Options{
		Help:     "defines the maximum size of a message",
		Required: true,
	})
	// minimum amount of events to be included in the experiment
	simNSplits := commandSim.Int("o", "nsplits", &argparse.Options{
		Help:     "number of encounters where split is needed",
		Required: true,
	})
	//time step value
	timeStep := commandSim.Float("z", "tstep", &argparse.Options{
		Help:     "defines the time step of epochs",
		Required: true,
	})
	//encoutner isolate time
	// encIso := commandSim.Float("l", "enciso", &argparse.Options{
	// 	Help:     "defines the minimum isolation time between consecutive encounters",
	// 	Required: true,
	// })
	// command line (CLI) args
	commandCLI := parser.NewCommand("cli", "use command-line interface")
	commandListLenses := commandCLI.NewCommand("lenses", "list lenses")
	commandImport := commandCLI.NewCommand("import", "import data from a dataset")
	commandImportLens := commandImport.String("l", "lens", &argparse.Options{
		Help:     "lens to use (" + lens.GetInstalledLenses() + ")",
		Required: true,
	})
	commandImportPath := commandImport.String("p", "path", &argparse.Options{
		Help:     "file or path to import",
		Required: true,
	})
	commandImportDatasetName := commandImport.String("n", "name", &argparse.Options{
		Help:     "dataset name",
		Required: true,
	})

	////get the parameters from the config file
	args := GetConfiguration(os.Args[1], os.Args[2:])

	err = parser.Parse(strings.Split(args, " "))
	if err != nil {
		fmt.Fprint(os.Stderr, parser.Usage(err))
		os.Exit(1)
	}
	// set up logger for the system
	log = logger.New()
	l, err := logger.ParseLevel(*logLevel)
	if err != nil {
		fmt.Fprint(os.Stderr, parser.Usage(err))
		os.Exit(1)
	}
	log.SetLevel(l)
	customFormatter := new(logger.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.000"
	log.SetFormatter(UTCFormatter{customFormatter})
	customFormatter.FullTimestamp = true

	log.Info("running with args ", os.Args)

	// set up PRNG
	model.Seed(int64(*seed))
	log.Infof("random seed is %v", *seed)
	log.Info("starting")
	log.Infof("logging at log level %v; all times in UTC", *logLevel)
	if path, err := os.Getwd(); err != nil {
		log.Fatalf("cannot get working directory: %v", err)
	} else {
		log.Debugf("running from %v", path)
	}

	// initialize the lenses
	lens.LensInit(log)
	// initialize the DB
	model.Init(log, *dbType, *dbFile)
	//init the counter of messages
	logics.InitCounter(log)
	//init the encounter manager - now constant of 20 seconds
	InitEncManager(*timeStep)
	// the ordering of the cases in this switch is extremely important; this
	// needs to be ordered by most specific command

	switch {
	case commandImport.Happened():
		if err := lens.Import(*commandImportLens, *commandImportPath, *commandImportDatasetName); err != nil {
			log.Warnf("import raised an error: %v", err)
		}
	case commandSim.Happened():
		//init memory control
		log.Infof("defining default memory/buffer")
		logics.InitMemory(log, *simMinBuffer)
		// initialize the logic engines in general
		//and the specific engine used
		logics.LogicEnginesInit(log, *simLogicFile)
		//run simulation func
		simulate(*simLogic, *simMessages, *simConditions, *simTimeStart, *simTimeEnd, *simDataset, *simExp, *simInvestigator, *simMaxBuffer, *simMinBuffer, float32(*simMinMesSize), float32(*simMaxMesSize), int(*simNSplits), float32(*timeStep), *simMessagesGen, *simMessagesTemp, *simGeneratorScript, *dbFile)
	case commandListLenses.Happened():
		lens.PrintLenses()
	case commandCLI.Happened():
		log.Debug("using command-line interface")
	case commandWeb.Happened():

		webService(*webHost, *webPort)
	}
	log.Info("ending")
}
