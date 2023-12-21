package logic

import (
	"testing"
)

// General tests:
// test a distance
func TestSimpleDist(t *testing.T) {
	pos1 := [3]int{1, 2, 3}
	pos2 := [3]int{1, 2, 3}
	gridSize := 20.0
	result := Distance(pos1, pos2, gridSize)
	expected := 0.0
	if result != expected {
		t.Errorf("Distance 1 failed. Expected: %f, got: %f", expected, result)
	}
}

func TestFarDist(t *testing.T) {
	pos1 := [3]int{100, 200, 300}
	pos2 := [3]int{1, 2, 3}
	gridSize := 20.0
	result := Distance(pos1, pos2, gridSize)
	expected := 7408.481625812404
	if result != expected {
		t.Errorf("Distance 2 failed. Error in Calculation")
	}
}

// func TestDataPreparation(t *testing.T) {
// 	// data preperation
// 	rows, err := model.DB.Table("events").Where("dataset_name=? and time>=? and time<=?", "napa", 1408000000, 1408500000).Order("time").Rows()
// 	if err != nil {
// 		logg.Infof("error in fetching the events for profiling %v", rows)
// 	}
// 	//DataPreparationGeneral
// }
