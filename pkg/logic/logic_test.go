package logic

import (
	// "bytes"
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

/*
func TestEncryptDecrypt(t *testing.T) {
	// a message
	tmp_mes := Message{
		MessageId:       "123",
		Sender:          1,
		Source:          1,
		Type:            AddressTypeUnicast,
		Destination:     1,
		DestinationNode: 1,
		Payload:         "aaa",
		CreationTime:    111111111,
		path:            nil,
		ShardsAvailable: 0,
		ShardID:         1,
		MShards:         true,
		FakeMessage:     false,
		TTLHops:         10,
		TTLTime:         10,
		LatHops:         10,
		Size:            11.11,
	}
	key, err1 := generateKey() //[]byte{1, 2, 3, 4, 4}
	if err1 != nil {
		t.Errorf("could not create key")

	}
	new_mes, err := encryptMessage(key, &tmp_mes)
	if err != nil {
		t.Errorf("could not encrypt the message: %v", tmp_mes.String())

	}
	dec_mes, err2 := decryptMessage(key, new_mes)
	if err2 != nil {
		t.Errorf("could not decrypt the message: %v", err2)
		//t.Errorf("could not decrypt the message: %v", dec_mes.String())
	}
	result := tmp_mes.Compare(*dec_mes)
	if result != true {
		t.Errorf("Distance 1 failed. Expected: %v, got: %v", tmp_mes.String(), dec_mes.String())
	}
}


func TestKey(t *testing.T) {

//get a key
key, err1 := generateKey() //[]byte{1, 2, 3, 4, 4}
if err1 != nil {
	t.Errorf("could not create key")
	
}
//split the key
parts, err := splitKey(key, 5, 3)
if err != nil {
		t.Errorf("key split problem: %v", err)
		
	}
	
	k2, err2 := reconstructKey(parts)
	if err2 != nil {
		t.Errorf("key recon problem: %v", err2)
	}
	if !bytes.Equal(key, k2) {
		t.Errorf("not the same keys")
	}
}
*/