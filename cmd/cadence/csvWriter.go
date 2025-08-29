package main

import (
	"encoding/csv"
	"fmt"
	model "marathon-sim/datamodel"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)
type countMessages = struct{
	Experiment      string
	Nodes            string
	Counts           string
}
type transferNodes struct {
	Experiment string
	Times            string
	Counts           string
}
type datasetExperimentsData = struct{
	Datasetname string
	Experiment string
	DistanceCondition string
}
func exportDatasetsExperiments(experiments []datasetExperimentsData, tableName string, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file for %s: %v", tableName, err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(experiments[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Datasets and experiments: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}

	for _, experiment := range experiments {
		row, err := convertToString(experiment)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}
	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}
//Export the report of numbers of encounters grouped by unique experiments
func exportEncounteredNodesCSV(tableName string, filePath string) error {
	var nodes []model.EncounteredNodes
	encounters, err := model.GetDatesetsAndEncounters()
	if err != nil {
		return fmt.Errorf("error querying table %s: %v", tableName, err)
	} else {
		for _, encounter:= range encounters{
			if r := model.DB.Find(&nodes, "experiment_name=?", encounter.Experiment); r.Error != nil {
				return fmt.Errorf("error finding encounteredNodes %s: %v", tableName, err)
			}
		}

		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("unable to crete csv file: %v", err)
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
		
		encounter := model.EncounteredNodes{}
		header, err := fieldNames(encounter)
		if err != nil {
			return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
		}
		
		if err := writer.Write(header); err != nil{
			return fmt.Errorf("error writing header: %v", err)
		}

		for _, node := range nodes {
			row, err := convertToString(node)
			if err != nil {
				return fmt.Errorf("error: %v", err)
			}

			if err := writer.Write(row); err != nil {
				return fmt.Errorf("error writing row: %v", err)
			}
		}
	}
	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}
//Export the report of encounters' locations grouped by unique dataset and experiments
func exportEncounters(tableName string, filePath string) error {
	type DatasetDistances struct {
		DatasetName string
		Distance    float64
		ExperimentName string
	}
	var datasetDistances []DatasetDistances
	fetchDataDistance := model.DB.Table("encounters").
		Select("distinct dataset_name, distance, experiment_name").
		Find(&datasetDistances)
	if fetchDataDistance.Error != nil {
		return fmt.Errorf("error finding encounteredNodes %s: %v", tableName, fetchDataDistance.Error)
	}
	var encounters []model.Encounter
	for _, conditions := range(datasetDistances){
		result := model.DB.Table("encounters").
		Where("dataset_name=? and distance=? and experiment_name=?", conditions.DatasetName, conditions.Distance, conditions.ExperimentName).
		Find(&encounters)
		
		if result.Error != nil {
			log.Warnf("error when performing query: %v", result.Error)
			return fmt.Errorf("error querying table %s: %v", tableName, result.Error)
		}
	}
	
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(encounters[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}

	for _, encounter := range encounters {
		row, err := convertToString(encounter)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}
	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}

func exportEvents(datasets []string, tableName string, filePath string) error {
	type eventrows = struct{
		Datasetname string
		Nodes string
		Count string
	}
	var events []eventrows
	/*datasets, err := model.GetDatasets()
	if err != nil || len(datasets) < 1 {
		return fmt.Errorf("error finding events %s: %v", tableName, err)
	}*/
	for _, dataset := range(datasets){
		rows, err := model.DB.Table("events").Select("dataset_name, node as Node,count(*) as NumEvents").Where("dataset_name=?", dataset).Group("dataset_name, node").Order("NumEvents DESC").Rows()
		if err != nil {
			return fmt.Errorf("error in table %s: %v", tableName, err)
		}
		defer rows.Close()
		for rows.Next() {
			var d, n, e string
			var event eventrows
			rows.Scan(&d, &n, &e)
			event.Datasetname = d
			event.Nodes = n
			event.Count = e
			events = append(events, event)
		}
	}
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(events[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}

	for _, event := range events {
		row, err := convertToString(event)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}
	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}

func exportEventLocation(datasets []string, tableName string, filePath string) error {
	type eventLocations = struct{
		Datasetname string
		Nodes string
		Count string
		Average_x float64
		Average_y float64
		Average_z float64
	}
	var eventLocationsRows []eventLocations
	/*datasets, err := model.GetDatasets()
	if err != nil || len(datasets) < 1 {
		return fmt.Errorf("error finding events‘ location %s: %v", tableName, err)
	}*/
	for _, dataset := range(datasets){
		rows, err := model.DB.Table("events").
			Select("dataset_name, node, count(*) as Count, avg(x) as avg_x, avg(y) as avg_y, avg(z) as avg_z").
			Where("dataset_name=?", dataset).Group("dataset_name, node").Rows()
		if err != nil {
			return fmt.Errorf("error finding events‘ location %s: %v", tableName, err)
		}
		defer rows.Close()
		for rows.Next() {
			var d, n, c string
			var x, y, z float64
			var event eventLocations
			rows.Scan(&d, &n, &c, &x, &y, &z)
			event.Datasetname = d
			event.Nodes = n
			event.Count = c
			event.Average_x = x
			event.Average_y = y
			event.Average_z = z

			eventLocationsRows = append(eventLocationsRows, event)
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(eventLocationsRows[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}

	for _, eventLocation:= range eventLocationsRows {
		row, err := convertToString(eventLocation)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}
	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}

func exportNodesTimeSpan(datasets []string, tableName string, filePath string) error{
	type timeSpanData =struct{
		Dataset      string
		Nodes         string
		Spans         string
	}
	var nodesTimeSpan []timeSpanData
	for _, dataset := range(datasets){
		rows, err := model.DB.Table("events").Select("dataset_name, node, min(time),max(time) as Span").Where("dataset_name=?", dataset).Group("dataset_name, node").Order("node").Rows()
		if err != nil {
			return fmt.Errorf("error fetching table %s: %v", tableName, err)
		}
		defer rows.Close()
		for rows.Next() {
			var nodeSpan timeSpanData
			var d, n string
			var minT, maxT float64
			if err := rows.Scan(&d, &n, &minT, &maxT); err != nil {
				return fmt.Errorf("error calculating data: %v", err)
			}
			nodeSpan.Dataset = d
			nodeSpan.Nodes = n
			spanString := fmt.Sprintf("[%v,%v]", minT, maxT)
			nodeSpan.Spans = spanString
			nodesTimeSpan = append(nodesTimeSpan, nodeSpan)
	}}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(nodesTimeSpan[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}

	for _, nodeSpan:= range nodesTimeSpan {
		row, err := convertToString(nodeSpan)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}
	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}
func exportSentMessages(experiments []string, tableName string, filePath string) error{
	var sentMessagesByNodes []countMessages
	for _, experiment := range(experiments){
		rows, err := model.DB.Table("message_dbs").Select("experiment_name, sender_node,count(reciever_node)").Where("experiment_name=?", experiment).Group("experiment_name, sender_node").Order("count(reciever_node)").Rows()
		if err != nil {
			return fmt.Errorf("error fetching %s data : %v", tableName, err)
		}
		defer rows.Close()

		var sentMessage countMessages
		for rows.Next() {
			var e, n string
			var amount string
			if err := rows.Scan(&e, &n, &amount); err != nil {
				return fmt.Errorf("cannot read in values: %v", err)
			}
			sentMessage.Experiment = e
			sentMessage.Nodes = n
			sentMessage.Counts = amount
			sentMessagesByNodes = append(sentMessagesByNodes, sentMessage)
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(sentMessagesByNodes[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}
	for _, sentMessagesByNode:= range sentMessagesByNodes {
		row, err := convertToString(sentMessagesByNode)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}

	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}

func exportReceivedtMessages(experiments []string, tableName string, filePath string) error{
	var receivedMessagesByNodes []countMessages
	for _, experiment := range(experiments){
		rows, err := model.DB.Table("message_dbs").Select("experiment_name, reciever_node,count(sender_node)").Where("experiment_name=?", experiment).Group("experiment_name, reciever_node").Order("count(sender_node)").Rows()
		if err != nil {
			return fmt.Errorf("error fetching %s data : %v", tableName, err)
		}
		defer rows.Close()

		var receivedMessage countMessages
		for rows.Next() {
			var e, n string
			var amount string
			if err := rows.Scan(&e, &n, &amount); err != nil {
				return fmt.Errorf("cannot read in values: %v", err)
			}
			receivedMessage.Experiment = e
			receivedMessage.Nodes = n
			receivedMessage.Counts = amount
			receivedMessagesByNodes = append(receivedMessagesByNodes, receivedMessage)
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(receivedMessagesByNodes[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}
	for _, receivedMessagesByNode:= range receivedMessagesByNodes {
		row, err := convertToString(receivedMessagesByNode)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}

	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}

func exportTransferredMessages(experiments []string, tableName string, filePath string) error{
	
	var transferByNodes []transferNodes
	for _, experiment := range(experiments){
		rows, err := model.DB.Table("message_dbs").Select("transfer_time").Where("experiment_name=?", experiment).Rows()
		if err != nil {
			return fmt.Errorf("error fetching %s data : %v", tableName, err)
		}
		defer rows.Close()

		times := make([]float64, 0)
		var transferByNode transferNodes
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				return fmt.Errorf("error %v", err)
			}
			//transferByNode.Experiment = e

			// Convert the time to float
			float_t, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return fmt.Errorf("error %v", err)
			}
			times = append(times, float_t)
		}
		rows.Close()

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
		for i := 0; i < len(labels); i++ {
			transferByNode.Times = labels[i]
			transferByNode.Experiment = experiment
			transferByNode.Counts = counts[i]
			transferByNodes = append(transferByNodes, transferByNode)
		
	
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(transferByNodes[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}
	for _, transferByNode:= range transferByNodes {
		row, err := convertToString(transferByNode)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}

	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}

func exportDeliveredMessages(experiments []string, tableName string, filePath string) error{
	
	var deliveredByNodes []transferNodes
	for _, experiment := range(experiments){
		rows, err := model.DB.Table("delivered_message_dbs").Select("deliver_time").Where("experiment_name=?", experiment).Rows()
		if err != nil {
			return fmt.Errorf("error fetching %s data : %v", tableName, err)
		}
		defer rows.Close()
		var start_delta float64
		start_delta = 1200000000
		if strings.Contains(experiment, "cabspotting") {
			start_delta = 1211000104
		}
		if strings.Contains(experiment, "hamburg") {
			start_delta = 1384927884
		}
		times := make([]float64, 0)
		var deliveredByNode transferNodes
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				return fmt.Errorf("error %v", err)
			}
			//transferByNode.Experiment = e

			// Convert the time to float
			float_t, err := strconv.ParseFloat(t, 64)
			if err != nil {
				return fmt.Errorf("error %v", err)
			}
			times = append(times, float_t+start_delta)
		}
		rows.Close()

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
		for i := 0; i < len(labels); i++ {
			deliveredByNode.Times = labels[i]
			deliveredByNode.Experiment = experiment
			deliveredByNode.Counts = counts[i]
			deliveredByNodes = append(deliveredByNodes, deliveredByNode)
		
	
		}
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(deliveredByNodes[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}
	for _, deliveredByNode:= range deliveredByNodes {
		row, err := convertToString(deliveredByNode)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}

	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}

func exportDeliveredRate(experiments []string, tableName string, filePath string) error{
	type data struct {
		Experiment     string
		DeliverRate    int64
	}
	var dataList []data
	for _, chosenExperiment := range experiments {
		var experiment data
		experiment.Experiment = chosenExperiment

		var transfer_count int64
		var deliver_count int64
		err := model.DB.Table("message_dbs").Select("count(distinct(message_id))").Where("experiment_name=?", chosenExperiment).Count(&transfer_count).Error
		if err != nil {
			return fmt.Errorf("error fetching %s data : %v", tableName, err)
		}

		err = model.DB.Table("delivered_message_dbs").Select("count(distinct(message_id))").Where("experiment_name=?", chosenExperiment).Count(&deliver_count).Error
		if err != nil {
			return fmt.Errorf("error fetching %s data : %v", tableName, err)
		}
		if transfer_count == 0 {
			experiment.DeliverRate = 0
		} else {
			experiment.DeliverRate = deliver_count * 100 / transfer_count
		}
		dataList = append(dataList, experiment)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("unable to crete csv file: %v", err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	header, err := fieldNames(dataList[0])
	if err != nil {
		return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
	}
		
	if err := writer.Write(header); err != nil{
		return fmt.Errorf("error writing header: %v", err)
	}
	for _, deliverRate:= range dataList {
		row, err := convertToString(deliverRate)
		if err != nil {
			return fmt.Errorf("error: %v", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing row: %v", err)
		}
	}

	fmt.Printf("CSV exported successfully to %s\n", filePath)
	return nil
}
func exportResultMetrics(tableName string, filePath string) error {
	var through model.ResultsDB
	var throughResult []model.ResultsDB
	rows, err := model.DB.Table("results_dbs").Rows()
		if err != nil {
			return fmt.Errorf("error fetching %s data : %v", tableName, err)
		}
		for rows.Next() {
			if err := rows.Scan(&through.ExperimentName, &through.LatSec,
				&through.LatHop, &through.MaxBuf, &through.MaxBand,
				&through.NetLoad, &through.Throughput, &through.NumMessages,
				&through.AvgCopiesMessages, &through.PeakLoad,
				&through.AverageLoad); err != nil {
					return fmt.Errorf("error in %s data : %v", tableName, err)

			}
			throughResult = append(throughResult, through)
		}
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("unable to crete csv file: %v", err)
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
	
		header, err := fieldNames(throughResult[0])
		if err != nil {
			return fmt.Errorf("unable to find header of Encountered Nodes: %v", err)
		}
			
		if err := writer.Write(header); err != nil{
			return fmt.Errorf("error writing header: %v", err)
		}
		for _, through:= range throughResult {
			row, err := convertToString(through)
			if err != nil {
				return fmt.Errorf("error: %v", err)
			}
	
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("error writing row: %v", err)
			}
		}
	
		fmt.Printf("CSV exported successfully to %s\n", filePath)
		return nil
}
func convertToString(obj interface{}) ([]string, error) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem() // Dereference if it's a pointer
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, got %s", v.Kind())
	}

	var result []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		// Handle nested structs (e.g., NodeId)
		if field.Kind() == reflect.Struct {
			nestedFields, err := convertToString(field.Interface())
			if err != nil {
				return nil, err
			}
			result = append(result, nestedFields...)
			continue
		}

		// Convert field value to string
		var value string
		switch field.Kind() {
		case reflect.String:
			value = field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			value = strconv.FormatInt(field.Int(), 10)
		case reflect.Float32, reflect.Float64:
			value = strconv.FormatFloat(field.Float(), 'f', 2, 64)
		case reflect.Bool:
			value = strconv.FormatBool(field.Bool())
		default:
			value = fmt.Sprintf("%v", field.Interface()) // Generic fallback
		}

		result = append(result, value)
	}

	return result, nil
}

func fieldNames(obj interface{}) ([]string, error){
	t := reflect.TypeOf(obj)
	var fieldNames []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldNames = append(fieldNames, field.Name)
	}
	return fieldNames, nil
}

func exportCSV(){
	datasets, err := model.GetDatasets()
	if err != nil || len(datasets) < 1 {
		fmt.Printf("Error: %v\n", err)
		return
	}
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	experimentNames := make([]string, 0, len(experiments))
	
	var dataExperimentsData []datasetExperimentsData
	for _, e := range experiments {
		experimentNames = append(experimentNames, e.ExperimentName)
		var dataExperiment datasetExperimentsData
		dataExperiment.Datasetname = e.DatasetName
		dataExperiment.Experiment = e.ExperimentName
		dataExperiment.DistanceCondition = e.DistanceConditions
		dataExperimentsData = append(dataExperimentsData, dataExperiment)
	}

	folderName := "csv"

	// Create the folder
	errDirect := os.Mkdir(folderName, os.ModePerm)
	if errDirect != nil {
		// Handle the error if the folder already exists or there's another issue
		if os.IsExist(errDirect) {
			fmt.Printf("Folder %s already exists\n", folderName)
		} else {
			fmt.Printf("Error creating folder: %v\n", errDirect)
			return
		}
	}

	encounteredNodesFileName := filepath.Join(folderName,"encounteredNodes.csv")
	if errEncounteredNodes := exportEncounteredNodesCSV("encounteredNodes", encounteredNodesFileName); errEncounteredNodes != nil {
		fmt.Printf("Error: %v\n", errEncounteredNodes)
	} else {
		fmt.Println("CSV file created successfully.")
	}
	encounterFileName := filepath.Join(folderName,"encounters.csv")
	if errEncounters := exportEncounters("encounters", encounterFileName); errEncounters != nil {
		fmt.Printf("Error: %v\n", errEncounters)
	} else {
		fmt.Println("CSV file created successfully.")
	}
	//The following csv table takes long time to generate, unquote them if necessary
	/*
	eventsNodesFileName := filepath.Join(folderName,"eventsByNodes.csv")
	if errEventsNodes := exportEvents(datasets,"eventsByNodes", eventsNodesFileName); errEventsNodes != nil {
		fmt.Printf("Error: %v\n", errEventsNodes)
	} else {
		fmt.Println("CSV file created successfully.")
	}

	eventsLocationFileName := filepath.Join(folderName,"eventsLocation.csv")
	if errEventsLocation := exportEventLocation(datasets, "eventsLocation", eventsLocationFileName); errEventsLocation != nil {
		fmt.Printf("Error: %v\n", errEventsLocation)
	} else {
		fmt.Println("CSV file created successfully.")
	}
	
	nodeTimeSpanFileName := filepath.Join(folderName,"nodeTimeSpan.csv")
	if errNodeTimeSpan := exportNodesTimeSpan(datasets, "nodeTimeSpan", nodeTimeSpanFileName); errNodeTimeSpan != nil {
		fmt.Printf("Error: %v\n", errNodeTimeSpan)
	} else {
		fmt.Println("CSV file created successfully.")
	}
	*/
	sentMessages := filepath.Join(folderName,"sentMessages.csv")
	if errSentMessages := exportSentMessages(experimentNames, "sentMessages", sentMessages); errSentMessages != nil {
		fmt.Printf("Error: %v\n", errSentMessages)
	} else {
		fmt.Println("CSV file created successfully.")
	}
	receivedMessages := filepath.Join(folderName,"receivedMessages.csv")
	if errReceivedMessages := exportReceivedtMessages(experimentNames, "receivedMessages", receivedMessages); errReceivedMessages != nil {
		fmt.Printf("Error: %v\n", errReceivedMessages)
	} else {
		fmt.Println("CSV file created successfully.")
	}


	transferredMessages := filepath.Join(folderName,"transferredMessages.csv")
	if errTransferredtMessages := exportTransferredMessages(experimentNames, "transferredMessages", transferredMessages); errTransferredtMessages != nil {
		fmt.Printf("Error: %v\n", errTransferredtMessages)
	} else {
		fmt.Println("CSV file created successfully.")
	}

	deliveredMessages := filepath.Join(folderName,"deliveredMessages.csv")
	if errdeliveredMessages := exportDeliveredMessages(experimentNames, "deliveredMessages", deliveredMessages); errdeliveredMessages != nil {
		fmt.Printf("Error: %v\n", errdeliveredMessages)
	} else {
		fmt.Println("CSV file created successfully.")
	}

	deliverRate := filepath.Join(folderName,"deliverRate.csv")
	if errDeliverRate := exportDeliveredRate(experimentNames, "deliverRate", deliverRate); errDeliverRate != nil {
		fmt.Printf("Error: %v\n", errDeliverRate)
	} else {
		fmt.Println("CSV file created successfully.")
	}

	resultMetricsTable := filepath.Join(folderName,"resultMetricsTable.csv")
	if errResultMetricsTable := exportResultMetrics("resultMetricsTable", resultMetricsTable); errResultMetricsTable != nil {
		fmt.Printf("Error: %v\n", errResultMetricsTable)
	} else {
		fmt.Println("CSV file created successfully.")
	}

	datasetExperiment := filepath.Join(folderName,"datasetExperiment.csv")
	if errDatasetExperiment := exportDatasetsExperiments(dataExperimentsData,"resultMetricsTable", datasetExperiment); errDatasetExperiment != nil {
		fmt.Printf("Error: %v\n", errDatasetExperiment)
	} else {
		fmt.Println("CSV file created successfully.")
	}
}