package main

import (
	//"bytes"
	"fmt"
	"strconv"
	//"image/png"
	//"log"
	model "marathon-sim/datamodel"
	"os"

	//"marathon-sim/lens"
	//logics "marathon-sim/logic"

	"github.com/jung-kurt/gofpdf"
	"github.com/wcharczuk/go-chart"
)
type nodeCount = struct{
	Nodes         string
	Counts        string
}
type timeCount = struct{
	Times            string
	Counts           string
}
func downloadReport() {
	//Data Analysis Prep
	datasetsData, err := getDatasetReport()
	if err != nil {
		log.Fatalf("Failed to fetch datasets data: %v", err)
	}

	experiments, err := getExperimentsReport()
	if err != nil {
		log.Fatalf("Failed to fetch experiments data: %v", err)
	}

	datasetEncounters, err := getEncounterReport()
	if err != nil {
		log.Fatalf("Failed to fetch datasetEncounters data: %v", err)
	}

	events, err := getEventReport()
	if err != nil {
		log.Fatalf("Failed to fetch events data: %v", err)
	}
	
	sentMsgs, err := getSentReport()
	if err != nil {
		log.Fatalf("Failed to fetch sentMsgs data: %v", err)
	}

	receivedMsgs, err := getReceivedReport()
	if err != nil {
		log.Fatalf("Failed to fetch receivedMsgs data: %v", err)
	}
	// Generate and save the PDF
	pdfFilename := "experiment_analysis.pdf"
	if err := getReport(datasetsData, experiments, pdfFilename, datasetEncounters, events, sentMsgs, receivedMsgs); err != nil {
		log.Fatalf("Failed to generate PDF: %v", err)
	}

	fmt.Printf("PDF report saved as '%s'\n", pdfFilename)
}
func getReport(datasets []string, experiments []model.Experiment,filename string, datasetEncounters map[string][]model.EncounteredNodes, events map[string][]nodeCount, sentMsgs map[string][]nodeCount, receivedMsgs map[string][]nodeCount) error{
	//Create a pdf file
	pdf := gofpdf.New("P","mm","A4", "")
	pdf.SetFont("Arial", "", 10)
	pdf.AddPage()
	//Title
	pdf.Cell(40, 10, "Experiment Analysis Report")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 16)
	//Datasets Info
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, "Datasets")
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 10)
	if len(datasets) == 0 {
        pdf.Cell(0, 10, "No datasets were found.")
        pdf.Ln(10)
    } else {
		for _, record := range datasets {
			paragraph := fmt.Sprintf("The datasets we used for the experiments: %s.", record)
			pdf.MultiCell(0, 10, paragraph, "", "L", false)
			pdf.Ln(5)  
		}
	}
	//Experiments Info
	pdf.SetFont("Arial", "B", 14)
    pdf.Cell(0, 10, "Experiments")
    pdf.Ln(10)
    pdf.SetFont("Arial", "", 10)
	if len(experiments) == 0 {
        pdf.Cell(0, 10, "No experiments were found.")
        pdf.Ln(10)
    } else {
		for index, record := range experiments {
			paragraph := fmt.Sprintf("%d. Experiment Name: %s, Dataset Used: %s, Investigator: %s, Date Started: %s, Date Finished: %s, Distance Conditions: %s ", index+1, record.ExperimentName, record.DatasetName, record.Investigator, record.DateStarted.Time.Format("2006-01-02"), record.DateFinished.Time.Format("2006-01-02"), record.DistanceConditions)
			pdf.MultiCell(0, 10, paragraph, "", "L", false)
			pdf.Ln(5) 
		}
	}
	//Encountered Nodes Count Info
	pdf.AddPage()
	addPage := 0
	// Section title
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, "Encounters for Dataset")
	pdf.Ln(10)
	for dataset, nodes := range datasetEncounters {
		if addPage != 0 && addPage % 2 == 0 {
			pdf.AddPage()
		}
		addPage += 1
		// Generate bar graph for this dataset
		chartPath := fmt.Sprintf("%s_encounters_bar_chart.png", dataset)
		createEncountersBarGraph(dataset, nodes, chartPath)

		// Add the bar chart to the PDF
		pdf.Image(chartPath, 10, pdf.GetY(), 180, 90, false, "", 0, "")
		pdf.Ln(95) // Add space after the chart

		// Delete the temporary chart file after use
		defer os.Remove(chartPath)
	}

	//Event Info
	addPage = 0
	// Section title
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, "Events for Dataset")
	pdf.Ln(10)
	for dataset, event := range events {
		if addPage != 0 && addPage % 2 == 0 {
			pdf.AddPage()
		}
		addPage += 1
		// Generate bar graph for this dataset
		chartPath := fmt.Sprintf("%s_events_bar_chart.png", dataset)
		createEventLineGraph(dataset, event, chartPath)

		// Add the bar chart to the PDF
		pdf.Image(chartPath, 10, pdf.GetY(), 180, 90, false, "", 0, "")
		pdf.Ln(95) // Add space after the chart

		// Delete the temporary chart file after use
		defer os.Remove(chartPath)
	}
	//Sent Messages by Nodes
	addPage = 0
	//Section Title
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10,"Messages Sent by Nodes")
	pdf.Ln(10)
	for experiment, sentNodes := range sentMsgs {
		if addPage != 0 && addPage % 2 == 0 {
			pdf.AddPage()
		}
		addPage += 1
		// Generate bar graph for this dataset
		chartPath := fmt.Sprintf("%s_sentNodes_bar_chart.png", experiment)
		createSentNodesBarGraph(experiment, sentNodes, chartPath)

		// Add the bar chart to the PDF
		pdf.Image(chartPath, 10, pdf.GetY(), 180, 90, false, "", 0, "")
		pdf.Ln(95) // Add space after the chart

		// Delete the temporary chart file after use
		defer os.Remove(chartPath)
	}

	//Received Messages by Nodes
	addPage = 0
	//Section Title
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10,"Messages Received by Nodes")
	pdf.Ln(10)
	for experiment, sentNodes := range sentMsgs {
		if addPage != 0 && addPage % 2 == 0 {
			pdf.AddPage()
		}
		addPage += 1
		// Generate bar graph for this dataset
		chartPath := fmt.Sprintf("%s_receivedNodes_bar_chart.png", experiment)
		createReceivedNodesBarGraph(experiment, sentNodes, chartPath)

		// Add the bar chart to the PDF
		pdf.Image(chartPath, 10, pdf.GetY(), 180, 90, false, "", 0, "")
		pdf.Ln(95) // Add space after the chart

		// Delete the temporary chart file after use
		defer os.Remove(chartPath)
	}

	// Save the PDF to a file
	err := pdf.OutputFileAndClose(filename)
	if err != nil {
		return err
	}
	fmt.Println("PDF generated successfully:", filename)
	return nil
}

func getDatasetReport()([]string, error){
	datasets, err := model.GetDatasets()
	if err != nil {
		return nil, err
	} else {
		return datasets, nil
	}
}

func getExperimentsReport()([]model.Experiment, error) {
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		return nil, err
	} else {
		return experiments, nil
	}
}

func getEncounterReport()(map[string][]model.EncounteredNodes, error){
	datasetData := make(map[string][]model.EncounteredNodes)
	encounters, err := model.GetDatesetsAndEncounters()
	if err != nil {
		return nil, err
	} else {
		for _, encounter:= range encounters{
			var nodes []model.EncounteredNodes
			if r := model.DB.Find(&nodes, "experiment_name=?", encounter.Experiment); r.Error != nil {
				log.Warnf("error when performing query: %v", err)
				return nil, r.Error
			}
			datasetData[encounter.Dataset] = append(datasetData[encounter.Dataset], nodes...)
		}
		return datasetData, nil
	}
}


func getEventReport()(map[string][]nodeCount, error){
	eventsdata := make(map[string][]nodeCount)
	datasets, err := model.GetDatasets()
	if err != nil || len(datasets) < 1 {
		return nil, err
	}
	for _, dataset := range(datasets){
		rows, err := model.DB.Table("events").Select("node as Node,count(*) as NumEvents").Where("dataset_name=?", dataset).Group("node").Order("NumEvents DESC").Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			return nil, err
		}
		defer rows.Close()
		events := make([]nodeCount, 0, 1000)
		for rows.Next() {
			var n, e string
			var event nodeCount
			rows.Scan(&n, &e)
			event.Nodes = n
			event.Counts = e
			events = append(events, event)
		}
		eventsdata[dataset] = append(eventsdata[dataset], events...)
	}
	return eventsdata, nil
}

func getSentReport()(map[string][]nodeCount, error){
	receivedNodeData := make(map[string][]nodeCount)
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		return nil, err
	}
	for _, experiment := range(experiments){
		rows, err := model.DB.Table("message_dbs").Select("sender_node,count(reciever_node)").Where("experiment_name=?", experiment.ExperimentName).Group("sender_node").Order("count(reciever_node)").Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			return nil, err
		}
		defer rows.Close()
		nodescounts := make([]nodeCount, 0, 1000)
		for rows.Next() {
			var n string
			var amount string
			var nodecount nodeCount
			if err := rows.Scan(&n, &amount); err != nil {
				log.Warnf("cannot read in values: %v", err)
				continue
			}
			nodecount.Nodes = n
			nodecount.Counts = amount
			nodescounts = append(nodescounts, nodecount)
		}
		receivedNodeData[experiment.ExperimentName] = append(receivedNodeData[experiment.ExperimentName], nodescounts...)

	}
	return receivedNodeData, nil

}
func getReceivedReport()(map[string][]nodeCount, error){
	sentNodeData := make(map[string][]nodeCount)
	experiments, err := model.GetDatasetsAndExperiments()
	if err != nil {
		return nil, err
	}
	for _, experiment := range(experiments){
		rows, err := model.DB.Table("message_dbs").Select("reciever_node,count(reciever_node)").Where("experiment_name=?", experiment.ExperimentName).Group("reciever_node").Order("count(sender_node)").Rows()
		if err != nil {
			log.Warnf("error when performing query: %v", err)
			return nil, err
		}
		defer rows.Close()
		nodescounts := make([]nodeCount, 0, 1000)
		for rows.Next() {
			var n string
			var amount string
			var nodecount nodeCount
			if err := rows.Scan(&n, &amount); err != nil {
				log.Warnf("cannot read in values: %v", err)
				continue
			}
			nodecount.Nodes = n
			nodecount.Counts = amount
			nodescounts = append(nodescounts, nodecount)
		}
		sentNodeData[experiment.ExperimentName] = append(sentNodeData[experiment.ExperimentName], nodescounts...)

	}
	return sentNodeData, nil

}

func createEncountersBarGraph(dataset string, nodes []model.EncounteredNodes, outputPath string) {
	var bars []chart.Value
	for _, node := range nodes {
		nodeName := strconv.Itoa(int(node.Node))
		bars = append(bars, chart.Value{Value: float64(node.Count), Label: nodeName})
	}

	// Create the bar chart
	barChart := chart.BarChart{
		Title: fmt.Sprintf("Encounters by Nodes for Dataset: %s", dataset),
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   50,  // Increased left padding to make room for the Y-axis
				Right:  20,
				Bottom: 60,  // Increased bottom padding to make room for the X-axis
			},
		},
		BarWidth:   20, // Width of each bar
		BarSpacing: 10, // Space between bars
		Width:      1200,  // Adjusted width for better fitting
		Height:     800,  // Adjusted height for better fitting
		XAxis: chart.Style{
			Hidden: false,
			FontSize: 6,
			FontColor: chart.ColorBlack,
			
			},
		YAxis: chart.YAxis{
			AxisType: chart.YAxisPrimary,
			Style: chart.Style{
				Hidden: false,
				FontSize: 10,
				FontColor: chart.ColorBlack,
			},
		},
		Bars: bars,
	}
	// Save the bar chart as a PNG file
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("failed to create bar chart file: %v", err)
	}
	defer f.Close()
	barChart.Render(chart.PNG, f)
}

func createEventLineGraph(dataset string, event []nodeCount, outputPath string) {
	var xLabels []string
	var yValues []float64

	// Convert event data to float64 and prepare the data for the line chart
	for _, e := range event {
		// Convert string (Events) to float64
		float_e, err := strconv.ParseFloat(e.Counts, 64)
		if err != nil {
			log.Fatalf("Error converting string to float64: %v", err)
		}
		// Add values for the line graph
		yValues = append(yValues, float_e)
		xLabels = append(xLabels, e.Nodes) // Use Nodes as X-axis labels
	}

	// Create the line chart
	lineChart := chart.Chart{
		Title: fmt.Sprintf("Events by Nodes for Dataset: %s", dataset),
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   60,  
				Right:  20,
				Bottom: 60,  
			},
		},
		Width:  1200,
		Height: 800,
		XAxis: chart.XAxis{
			Style: chart.Style{
				FontSize: 10,
			},
			ValueFormatter: func(v interface{}) string {
				idx := int(v.(float64)) - 1 
				if idx >= 0 && idx < len(xLabels) {
					if idx%3 == 0 {
						return xLabels[idx]
					}
				}
				return ""
			},
		},
		YAxis: chart.YAxis{
			AxisType: chart.YAxisPrimary,
			Style: chart.Style{
				FontSize:   10,
				FontColor:  chart.ColorBlack,
			},
		},
		Series: []chart.Series{
			// Add continuous series for the line chart
			chart.ContinuousSeries{
				XValues: generateIndexArray(len(yValues)), 
				YValues: yValues,                          
				Style: chart.Style{
					StrokeWidth: 2,               
					StrokeColor: chart.ColorBlue, 
				},
			},
		},
	}

	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("failed to create line chart file: %v", err)
	}
	defer f.Close()
	lineChart.Render(chart.PNG, f)
}
// helper function for line chart
func generateIndexArray(length int) []float64 {
	indices := make([]float64, length)
	for i := 0; i < length; i++ {
		indices[i] = float64(i + 1)
	}
	return indices
}

func createSentNodesBarGraph(experiment string, sentNodes []nodeCount, outputPath string) {
	var bars []chart.Value
	for _, node := range sentNodes {
		count, err := strconv.Atoi(node.Counts)
		if err != nil {
			fmt.Println("Error converting string to integer:", err)
			return
		}
		bars = append(bars, chart.Value{Value: float64(count), Label: node.Nodes})
	}

	// Create the bar chart
	barChart := chart.BarChart{
		Title: fmt.Sprintf("Sent Messages by Nodes for Dataset: %s", experiment),
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   50,  // Increased left padding to make room for the Y-axis
				Right:  20,
				Bottom: 60,  // Increased bottom padding to make room for the X-axis
			},
		},
		BarWidth:   20, // Width of each bar
		BarSpacing: 10, // Space between bars
		Width:      1200,  // Adjusted width for better fitting
		Height:     800,  // Adjusted height for better fitting
		XAxis: chart.Style{
			Hidden: false,
			FontSize: 6,
			FontColor: chart.ColorBlack,
			
			},
		YAxis: chart.YAxis{
			AxisType: chart.YAxisPrimary,
			Style: chart.Style{
				Hidden: false,
				FontSize: 10,
				FontColor: chart.ColorBlack,
			},
		},
		Bars: bars,
	}
	// Save the bar chart as a PNG file
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("failed to create bar chart file: %v", err)
	}
	defer f.Close()
	barChart.Render(chart.PNG, f)
}

func createReceivedNodesBarGraph(experiment string, receivedNodes []nodeCount, outputPath string) {
	var bars []chart.Value
	for _, node := range receivedNodes {
		count, err := strconv.Atoi(node.Counts)
		if err != nil {
			fmt.Println("Error converting string to integer:", err)
			return
		}
		bars = append(bars, chart.Value{Value: float64(count), Label: node.Nodes})
	}

	// Create the bar chart
	barChart := chart.BarChart{
		Title: fmt.Sprintf("Received Messages by Nodes for Dataset: %s", experiment),
		Background: chart.Style{
			Padding: chart.Box{
				Top:    20,
				Left:   50,  // Increased left padding to make room for the Y-axis
				Right:  20,
				Bottom: 60,  // Increased bottom padding to make room for the X-axis
			},
		},
		BarWidth:   20, // Width of each bar
		BarSpacing: 10, // Space between bars
		Width:      1200,  // Adjusted width for better fitting
		Height:     800,  // Adjusted height for better fitting
		XAxis: chart.Style{
			Hidden: false,
			FontSize: 6,
			FontColor: chart.ColorBlack,
			
			},
		YAxis: chart.YAxis{
			AxisType: chart.YAxisPrimary,
			Style: chart.Style{
				Hidden: false,
				FontSize: 10,
				FontColor: chart.ColorBlack,
			},
		},
		Bars: bars,
	}
	// Save the bar chart as a PNG file
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("failed to create bar chart file: %v", err)
	}
	defer f.Close()
	barChart.Render(chart.PNG, f)
}
