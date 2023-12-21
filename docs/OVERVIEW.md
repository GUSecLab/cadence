# Overview

In a nutshell, a simulation is composed from four parts:

1. Parsing the configuration files.
2. Building the encounters table (if needed).
3. Simulating message transferring.
4. Saving the results.


## Parsing the configuration files

In this part, we parse the configuration files.
Please consult the [CONFILES.md](CONFILES.md) for examples of each configuration file.
The most important file is the general configuration file. In this file, we define how the simulation work, where the important files are,etc.
Another important file is the logic conf file. It governs vital parts for each logic engine.
The parsing of the configuration files is done in the main.go file (mostly the general and logic conf files).
In this process, we also initilize several components of the simulator, such as the logic engines, counter for message copies, encounters generator, and other vital components.

## Building the encounters table
A simulation is run on a preprocessed encounters table. This table is produced from the dataset, based on a time_step and distance (see [CONFILES.md](CONFILES.md) ). The encounters are generated in a parllelized process using nodeworkers. 
During the construction of encounters, every two consecutive encounters (or more consecutive encounters) are merged to one long encounter.
After the construction of the encounters table, an additional table is created that signs the existance of encounters table, based on the dataset, distance, and time_step. Based on this table, called "encounters_create", future simulations that use these settings would not enforce the creation of the table again.

## Simulating message transferring
In this part, we first generate messages based on the encounters table. It is reasonable to use this table and not the original dataset, as nodes that do not encounters any other node would not be able to transfer any message.
Then, based on the logic engine, we simulate the message transfer.

## Saving the results
After the message transfering is terminated, we aggregate the results in the ResultsDB table:

```
type ResultsDB struct {
	ExperimentName    string  `gorm:"primaryKey"` //experiment name
	LatSec            float32 //latency in seconds (average)
	LatHop            float32 //latency in hops (average)
	MaxBuf            float32 //max buffer
	MaxBand           int     //max bandwidth
	NetLoad           int     //network load
	Throughput        float32 //ratio of delivered messages
	NumMessages       int     //number of messages in total
	AvgCopiesMessages float64 //average amount of copies of messages
	PeakLoad          float64 //the pick load during the experiment
	AverageLoad       float64 //average load during the experiment
}
```

