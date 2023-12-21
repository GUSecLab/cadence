package main

import (
	"encoding/json"
	"fmt"
	model "marathon-sim/datamodel"
	"marathon-sim/lens"
	logics "marathon-sim/logic"
	"math"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"
	"text/template"
)

var tmpl *template.Template

func exitHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("exit received")
	fmt.Fprint(w, "Program exiting.")
	os.Exit(0)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {

	// get the available lenses
	lensNames := make([]string, 0, len(lens.LensStore))
	for name := range lens.LensStore {
		lensNames = append(lensNames, name)
	}

	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := struct {
		Experiments *[]model.Experiment
		Logics      []string
		Lenses      []string
	}{
		Experiments: &experiments,
		Logics:      logics.GetInstalledLogicEngines(),
		Lenses:      lensNames,
	}

	if err := tmpl.ExecuteTemplate(w, "status.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

func datasetsHandler(w http.ResponseWriter, r *http.Request) {

	datasets, err := model.GetDatasets()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := struct {
		Datasets []string
	}{
		Datasets: datasets,
	}

	if err := tmpl.ExecuteTemplate(w, "datasets.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// generate a plot of the number of events per node
func numEventsByNodeHandler(w http.ResponseWriter, r *http.Request) {

	datasets, err := model.GetDatasets()
	if err != nil || len(datasets) < 1 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := struct {
		Datasets      []string
		ChosenDataset string
		Nodes         string
		Events        string
	}{
		Datasets: datasets,
	}

	chosenDataset := r.URL.Query().Get("dataset")
	if chosenDataset != "" {

		rows, err := model.DB.Table("events").Select("node as Node,count(*) as NumEvents").Where("dataset_name=?", chosenDataset).Group("node").Order("NumEvents DESC").Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		nodes := make([]string, 0, 1000)
		events := make([]string, 0, 1000)
		for rows.Next() {
			var n, e string
			rows.Scan(&n, &e)
			nodes = append(nodes, n)
			events = append(events, e)
		}

		data.ChosenDataset = chosenDataset
		data.Nodes = strings.Join(nodes, ",")
		data.Events = strings.Join(events, ",")

	}

	if err := tmpl.ExecuteTemplate(w, "eventsbynode.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// generate a plot of the timespan for each node
func timeSpanByNodeHandler(w http.ResponseWriter, r *http.Request) {

	datasets, err := model.GetDatasets()
	if err != nil || len(datasets) < 1 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := struct {
		Datasets      []string
		ChosenDataset string
		Nodes         string
		Spans         string
	}{
		Datasets: datasets,
	}

	chosenDataset := r.URL.Query().Get("dataset")
	if chosenDataset != "" {

		rows, err := model.DB.Table("events").Select("node,min(time),max(time) as Span").Where("dataset_name=?", chosenDataset).Group("node").Order("node").Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		nodes := make([]string, 0, 1000)
		spans := make([]string, 0, 1000)
		for rows.Next() {
			var n string
			var minT, maxT float64
			if err := rows.Scan(&n, &minT, &maxT); err != nil {
				log.Warnf("cannot read in values: %v", err)
				continue
			}

			spanString := fmt.Sprintf("[%v,%v]", minT, maxT)
			nodes = append(nodes, n)
			spans = append(spans, spanString)
		}

		data.ChosenDataset = chosenDataset
		data.Nodes = strings.Join(nodes, ",")
		data.Spans = strings.Join(spans, ",")

	}

	if err := tmpl.ExecuteTemplate(w, "spansbynode.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// generate a plot of the messages received by each node
func MessagesRecPerNodeHandler(w http.ResponseWriter, r *http.Request) {
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	experimentNames := make([]string, 0, len(experiments))
	for _, e := range experiments {
		experimentNames = append(experimentNames, e.ExperimentName)

	}
	data := struct {
		Experiments      []string
		ChosenExperiment string
		Nodes            string
		Counts           string
	}{
		Experiments: experimentNames,
	}

	chosenExperiment := r.URL.Query().Get("experiment")
	if chosenExperiment != "" {
		rows, err := model.DB.Table("message_dbs").Select("reciever_node,count(sender_node)").Where("experiment_name=?", chosenExperiment).Group("reciever_node").Order("count(sender_node)").Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		nodes := make([]string, 0, 1000)
		counts := make([]string, 0, 1000)
		for rows.Next() {
			var n string
			var amount string
			if err := rows.Scan(&n, &amount); err != nil {
				log.Warnf("cannot read in values: %v", err)
				continue
			}

			nodes = append(nodes, n)
			counts = append(counts, amount)
		}

		data.ChosenExperiment = chosenExperiment
		data.Nodes = strings.Join(nodes, ",")
		data.Counts = strings.Join(counts, ",")

	}

	if err := tmpl.ExecuteTemplate(w, "messagespernode.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// generate a plot messages sent by each node
func MessagesPerNodeHandler(w http.ResponseWriter, r *http.Request) {
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	experimentNames := make([]string, 0, len(experiments))
	for _, e := range experiments {
		experimentNames = append(experimentNames, e.ExperimentName)

	}
	data := struct {
		Experiments      []string
		ChosenExperiment string
		Nodes            string
		Counts           string
	}{
		Experiments: experimentNames,
	}

	chosenExperiment := r.URL.Query().Get("experiment")
	if chosenExperiment != "" {
		rows, err := model.DB.Table("message_dbs").Select("sender_node,count(reciever_node)").Where("experiment_name=?", chosenExperiment).Group("sender_node").Order("count(reciever_node)").Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		nodes := make([]string, 0, 1000)
		counts := make([]string, 0, 1000)
		for rows.Next() {
			var n string
			var amount string
			if err := rows.Scan(&n, &amount); err != nil {
				log.Warnf("cannot read in values: %v", err)
				continue
			}

			nodes = append(nodes, n)
			counts = append(counts, amount)
		}

		data.ChosenExperiment = chosenExperiment
		data.Nodes = strings.Join(nodes, ",")
		data.Counts = strings.Join(counts, ",")

	}

	if err := tmpl.ExecuteTemplate(w, "messagesrecpernode.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// generate a plot of the messages received by each node
func MessageDeliveringTimeHandler(w http.ResponseWriter, r *http.Request) {
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	experimentNames := make([]string, 0, len(experiments))
	for _, e := range experiments {
		experimentNames = append(experimentNames, e.ExperimentName)

	}
	data := struct {
		Experiments      []string
		ChosenExperiment string
		Times            string
		Counts           string
	}{
		Experiments: experimentNames,
	}

	chosenExperiment := r.URL.Query().Get("experiment")
	if chosenExperiment != "" {
		rows, err := model.DB.Table("delivered_message_dbs").Select("deliver_time").Where("experiment_name=?", chosenExperiment).Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var start_delta float64 //shift from start of data
		log.Info(chosenExperiment)
		//based on the experiment
		start_delta = 1200000000
		if strings.Contains(chosenExperiment, "cabspotting") {
			start_delta = 1211000104
		}
		if strings.Contains(chosenExperiment, "hamburg") {
			start_delta = 1384927884
		}
		//convert the times to floats so we can count them
		times := make([]float64, 0)
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				log.Warnf("cannot read in values: %v", err)
				continue
			}
			// Convert the time to float
			float_t, err := strconv.ParseFloat(t, 64)
			if err != nil {
				fmt.Println("Error parsing float:", err)
				return
			}
			//add the shift to the data
			times = append(times, float_t+start_delta)

		}

		// Find the minimum value
		min := math.Inf(1)
		for _, v := range times {
			if v < min {
				min = v
			}
		}
		// Find the maximum value
		max := math.Inf(-1)
		for _, v := range times {
			if v > max {
				max = v
			}
		}

		labels, counts := createBuckets(times, min, max)

		data.ChosenExperiment = chosenExperiment
		data.Times = strings.Join(labels, ",")
		data.Counts = strings.Join(counts, ",")

	}

	if err := tmpl.ExecuteTemplate(w, "deliveredmessagetimesnode.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// this function generates a box & whiskers graph
// based on the requested metric
// generate a plot of the messages delivery for box and whiskers
func MetricsBoxes(w http.ResponseWriter, r *http.Request) {

	experimentsfamilies, err := model.GetExperimentsFamily()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//create a map of logic to slice of family members
	//this slice will be the base step for
	//the visualization of the families
	familyLogicToExperiments := make(map[string]map[string][]*model.ResultsDB, 0)
	for _, e := range experimentsfamilies {
		//get the throughput
		expname := e.ExperimentName
		// var throughput_cur float32
		var through model.ResultsDB
		var err error
		// rows, err := model.DB.Table("results_dbs").Select("throughput").Where("experiment_name=?", expname).Rows()
		// if err != nil {
		// 	log.Warnf("error when performing query of thorughput of experiment: %v, %v", expname, err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }
		rows, err := model.DB.Table("results_dbs").Where("experiment_name=?", expname).Rows()
		if err != nil {
			log.Warnf("error when performing query of resultsdb of experiment: %v, %v", expname, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for rows.Next() {
			if err := rows.Scan(&through.ExperimentName, &through.LatSec,
				&through.LatHop, &through.MaxBuf, &through.MaxBand,
				&through.NetLoad, &through.Throughput, &through.NumMessages,
				&through.AvgCopiesMessages, &through.PeakLoad,
				&through.AverageLoad); err != nil {
				log.Warnf("cannot read in results db value %v: %v", rows, err)
				continue
			}
			break
		}

		//aggreagate the current thorugput
		//to the logic
		key := e.FamilyDataset + "_" + strconv.FormatFloat(e.FamilyDistance, 'f', -1, 32)
		// Check if the inner map is already initialized.
		if familyLogicToExperiments[key] == nil {
			familyLogicToExperiments[key] = make(map[string][]*model.ResultsDB)
		}

		familyLogicToExperiments[key][e.Logic] = append(familyLogicToExperiments[key][e.Logic], through.Copy())

	}

	//add the title, the yaxis label and range
	// familyLogicToExperiments[title] = make(map[string][]float32)

	// familyLogicToExperiments[title][yaxis] = yrange

	// Convert data to JSON
	jsonData, err := json.Marshal(familyLogicToExperiments)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	escapedJSON := template.JSEscapeString(string(jsonData))
	// fmt.Printf("JSON Data: %s\n", escapedJSON)
	//execute the box and whiskers graph creation
	if err := tmpl.ExecuteTemplate(w, "metricsboxes.html", escapedJSON); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// generate a plot of the messages received by each node
func MessageTransfersPerTimeHandler(w http.ResponseWriter, r *http.Request) {
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	experimentNames := make([]string, 0, len(experiments))
	for _, e := range experiments {
		experimentNames = append(experimentNames, e.ExperimentName)

	}
	data := struct {
		Experiments      []string
		ChosenExperiment string
		Times            string
		Counts           string
	}{
		Experiments: experimentNames,
	}

	chosenExperiment := r.URL.Query().Get("experiment")
	if chosenExperiment != "" {
		rows, err := model.DB.Table("message_dbs").Select("transfer_time").Where("experiment_name=?", chosenExperiment).Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		//convert the times to floats so we can count them
		times := make([]float64, 0)
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				log.Warnf("cannot read in values: %v", err)
				continue
			}
			// Convert the time to float
			float_t, err := strconv.ParseFloat(t, 64)
			if err != nil {
				fmt.Println("Error parsing float:", err)
				return
			}
			times = append(times, float_t)

		}

		// Find the minimum value
		min := math.Inf(1)
		for _, v := range times {
			if v < min {
				min = v
			}
		}
		// Find the maximum value
		max := math.Inf(-1)
		for _, v := range times {
			if v > max {
				max = v
			}
		}
		labels, counts := createBuckets(times, min, max)

		data.ChosenExperiment = chosenExperiment
		data.Times = strings.Join(labels, ",")
		data.Counts = strings.Join(counts, ",")

	}

	if err := tmpl.ExecuteTemplate(w, "messgagetimesnode.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// create the buckets to count values in intervals
func createBuckets(values []float64, minValue float64, maxValue float64) ([]string, []string) {
	// Determine the range of values and the number of buckets required
	rangeValues := maxValue - minValue
	numBuckets := 10 //int(math.Ceil(rangeValues / bucketInterval))
	bucketInterval := rangeValues / float64(numBuckets)
	// Create the slices for the bucket labels and counts
	labels := make([]string, numBuckets)
	counts := make([]int, numBuckets)

	// Initialize the start value for the first bucket
	startValue := minValue

	// Iterate over the input values and increment the count of the corresponding bucket
	for _, value := range values {
		bucketIndex := math.Min((value-minValue)/bucketInterval, float64(numBuckets-1))
		bucketIndex = math.Max(bucketIndex, 0)
		counts[int(bucketIndex)]++
	}

	// Create the labels for the buckets
	for i := 0; i < numBuckets; i++ {
		endValue := startValue + bucketInterval
		labels[i] = fmt.Sprintf("'"+"%.f_%.f"+"'", startValue, endValue)
		startValue = endValue
	}

	//convert the counts to strings so it will be digested
	//in the graphing function
	// Convert the slice of integers to a slice of strings
	strs := make([]string, len(counts))
	for i, n := range counts {
		strs[i] = strconv.Itoa(n)
	}
	return labels, strs
}

// generate a plot of the number of encounters for each node
func encounterHandler(w http.ResponseWriter, r *http.Request) {

	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	experimentNames := make([]string, 0, len(experiments))
	for _, e := range experiments {
		experimentNames = append(experimentNames, e.ExperimentName)
	}

	data := struct {
		Experiments      []string
		ChosenExperiment string
		Nodes            string
		Counts           string
	}{
		Experiments: experimentNames,
	}

	chosenExperiment := r.URL.Query().Get("experiment")
	if chosenExperiment != "" {

		var encounteredNodes []model.EncounteredNodes

		if r := model.DB.Find(&encounteredNodes, "experiment_name=?", chosenExperiment); r.Error != nil {
			log.Warnf("error when performing query: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		nodes := make([]string, 0, len(encounteredNodes))
		counts := make([]string, 0, len(encounteredNodes))

		for _, en := range encounteredNodes {
			nodes = append(nodes, strconv.Itoa(int(en.Node)))
			counts = append(counts, strconv.Itoa(en.Count))
		}

		data.ChosenExperiment = chosenExperiment
		data.Nodes = strings.Join(nodes, ",")
		data.Counts = strings.Join(counts, ",")

	}

	if err := tmpl.ExecuteTemplate(w, "encounters.html", data); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// describes the experiments
func experimentsListHandler(w http.ResponseWriter, r *http.Request) {

	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "experiments.html", experiments); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

func reportsHandler(w http.ResponseWriter, r *http.Request) {
	if err := tmpl.ExecuteTemplate(w, "reports.html", nil); err != nil {
		log.Warnf("cannot execute template: %v", err)
	}
}

// handles web requests.  add new functionality here via the `http.HandleFunc`
// method
func webService(webHost string, webPort int) {
	var err error
	tmpl, err = template.ParseGlob("html/*.html")
	if err != nil {
		log.Fatalf("cannot load templates: %v", err)
	}

	http.HandleFunc("/profile", pprof.Profile)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/eventsByNode", numEventsByNodeHandler)
	http.HandleFunc("/spansByNode", timeSpanByNodeHandler)
	http.HandleFunc("/experimentsList", experimentsListHandler)
	http.HandleFunc("/encounters", encounterHandler)
	http.HandleFunc("/datasetsList", datasetsHandler)
	http.HandleFunc("/messagespernode", MessagesPerNodeHandler)
	http.HandleFunc("/messagesrecpernode", MessagesRecPerNodeHandler)
	http.HandleFunc("/messgagetimesnode", MessageTransfersPerTimeHandler)
	http.HandleFunc("/deliveredmessagetimesnode", MessageDeliveringTimeHandler)
	http.HandleFunc("/metricsboxes", MetricsBoxes)
	//http.HandleFunc("/deliveredmessagesbox", LatHopBox)

	http.HandleFunc("/exit", exitHandler)
	http.HandleFunc("/reports/", reportsHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("html/static"))))
	http.HandleFunc("/", reportsHandler)

	log.Infof("will listen on %v (port %v)", webHost, webPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%v:%d", webHost, webPort), nil))
}
