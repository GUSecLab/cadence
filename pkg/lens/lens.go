package lens

import (
	"database/sql"
	"errors"
	"fmt"
	model "marathon-sim/datamodel"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
)

// a Lens is a thing that reads in some dataset and imports it into the DB
type Lens interface {
	// initializes the lens
	Init(logger *logger.Logger)

	// imports a file (or files, if `path` is a directory) and updates the
	// `dataset` table
	Import(path string, dataSetName string) error

	// gets the type of location used for this dataset
	GetLocationType() model.LocationType
}

// a map of all registered lenses.  There should be one lens for each dataset
// file format
var LensStore map[string]Lens

// initialize the lenses; new lenses need to be added here
func LensInit(log *logger.Logger) {
	LensStore = make(map[string]Lens)

	// populate the LensStore with all of the available lenses
	// LensStore["randomWalk"] = &RandomWalk{}
	// LensStore["geolife"] = &Geolife{}
	// LensStore["cabspotting"] = &Cabspotting{}
	// LensStore["hamburg"] = &Hamburg{}
	LensStore["tdrive"] = &TDrive{}
	// LensStore["wipha"] = &Wipha{}
	// LensStore["kalmaegi"] = &Kalmaegi{}
	// LensStore["rammasun"] = &Rammasun{}
	// LensStore["halong"] = &Halong{}
	// LensStore["napa"] = &Napa{}
	// LensStore["mdc"] = &MDC{}
	// LensStore["simple"] = &SimpleDatasetFormat{}
	LensStore["japan"] = &Japan{}
	// LensStore["cabspotting2D"] = &Cabspotting2D{}
	// LensStore["geolife2D"] = &Geolife2D{}

	// initialize each lens
	for name, l := range LensStore {
		log.Debugf("initializing lens '%v'", name)
		l.Init(log)
	}
}

func GetInstalledLenses() string {
	lensesArr := make([]string, 0, len(LensStore))
	for k := range LensStore {
		lensesArr = append(lensesArr, k)
	}
	return strings.Join(lensesArr, ",")
}

func PrintLenses() {
	fmt.Println("installed lenses:")
	for lens := range LensStore {
		fmt.Println("\t" + lens)
	}
}

// dispatches the appropriate lens
func Import(config *model.Config) error {

	// extract the lens name, path, and dataset name from the config
	lensName := config.CLI.Lens
	path := config.CLI.Path
	dataSetName := config.CLI.Name


	l, ok := LensStore[lensName]
	if !ok {
		s := fmt.Sprintf("invalid lens name; valid lens names are [%v]", GetInstalledLenses())
		return errors.New(s)
	}
	logger.Infof("using lens %v to import data from '%v'; saving result as %v", lensName, path, dataSetName)
	ds := model.Dataset{
		DatasetName:    dataSetName,
		DateImported:   sql.NullTime{Valid: true, Time: time.Now()},
		CoordType:      l.GetLocationType(),
		CompleteImport: false,
	}

	// do the import
	if err := l.Import(path, dataSetName); err != nil {
		return err
	}
	
	// add record to DB (if import succeeds)
	if r := model.DB.Save(&ds); r.Error != nil {
		return r.Error
	}

	fmt.Println("Succeeded in importing ", dataSetName)
	// update the time
	ds.DateImported.Time = time.Now()
	ds.CompleteImport = true
	r := model.DB.Save(&ds)
	return r.Error
}
