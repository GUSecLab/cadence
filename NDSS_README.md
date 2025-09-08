# Artifact - The Cadence Simulator 

This repository contains the Cadence Simulator, the artifact for our paper "Mirage: Private, Mobility-based Routing for Censorship Evasion" published at NDSS 2026.  

The artifact is citable via the following DOI: [10.5281/zenodo.16953762](https://doi.org/10.5281/zenodo.16953762).

The Cadence Simulator provides a framework to test decentralized message passing algorithms on mobility datasets. For additional info, see [CADENCE.md](CADENCE.md).

# Structure 

- `cmd/cadence/` contains source code for the Cadence Simulator.
- `cmd/cadence/configs/` contains experiment configuration files.
- `docs/` contains additional documentation.
- `imgs/` contains media used for documentation.
- `marathon-mobility-data/` contains datasets used for Cadence simulations. 
- `message_files/` contains message files used for Cadence simulations. 
- `pkg/` contains additional source code for the Cadence Simulator. 
- `pkg/logic/logic_configs/` contains additional experiment configuration files
- `scripts/` contains python scripts to automate simulations. 
- `tools/` contains optional utilities such as sql scripts. 
- `tufte-css/` contains utilities for the Cadence web application interface. 
- `Dockerfile` contains docker commands
- `CADENCE.md` contains a general description of the Cadence Simulator. 
- `README.md` is this document, contains NDSS specific instructions for the Cadence Simulator. 
- `LICENSE` contains the license. 

- additionally, the scripts will generate four new directories:
- `db` stores the db files
- `results` stores simulation results
- `results/raw-data` stores raw data from simulation experiments in csv format
- `results/plots` stores plots generated from raw data

NOTE: if the scripts fail to generate db, results, results/raw-data, or results/plots directories, please create them manually to ensure successful execution!

# EXPERIMENT INSTRUCTIONS FOR NDSS 2026

All instructions assume a Linux operating system environment, although they should work on most systems since we are operating in a python virtual environment. 

Strongly recommend setting up a fresh virtual machine with Linux Ubuntu 24.04.2 LTS to ensure smooth operation. The scripts were run and tested on an Ubuntu 24.04.2 system with 8 GB ram and 25 GB disk space. 

*This repository has been dockerized - scroll to the end of this document for alternate docker instructions*

## setting up a virtual environment 

to ensure smooth execution, please set up a python virtual environment. 

(make sure python3 and pip are installed)

```
$ sudo apt update
$ sudo apt install python3 python3-pip
$ sudo apt install python3.12-venv
```

create a virtual environment and activate it

```
$ python3 -m venv venv
$ source venv/bin/activate
```

## Running the simulator 

There are two ways to run the simulator:

1. run this do-everything script, which will automate the entire process 
2. run each step one after the other, starting at step 1: setup. 

## do-everything script

This do-everything script simply runs each of the step scripts one after the other. It is recommended to run each step individually, in case any step creates problems, but the do-everything script should be able to automate all steps. 

This script will generate raw data and plots and store them in results/raw-data and results/plots respectively.

```
$ python3 scripts/run_all.py
```

* Note: if the do-everything script executes successfully, you do not need to run the following steps, as they have already been executed by the do-everything script!

## Step 1: setup 

To run cadence, you need the following things:

1. cadence executable (step 2)
2. golang
3. sqlite3
4. LaTex Packages (texlive)
5. various python modules for plotting (requirements.txt)

You can choose to install and setup these items yourself. Most of these should already be set up on most systems. You may also use this script to automate the process.

``` 
$ python3 scripts/run_setup.py
```

* Note: If go is not detected, the script will install go at /usr/local. If this is unacceptable, please install go manually and do not use the script. 
* Note: If sqlite3 is undetected, the script will install sqlite3 using the apt package manager. 
* Note: If LaTex packages are missing, the script will install using the apt package manager. 
* Note: You may have to add the path to the go installation manually to the path variable. 

```
$ export PATH=$PATH:/usr/local/go/bin
```

## Step 2: build the simulator 

This script will build the cadence executable 

```
$ python3 scripts/run_build.py
```

## Step 3: import datasets 

This script will set up two sqlite databases and import the japan and tdrive datasets respectively. 

```
$ python3 scripts/run_import.py
```

You may optionally choose to run the imports in parrallel using this version of the script

```
$ python3 scripts/run_import_parrallel.py
```

## Step 4: run experiments 

This script will run all relevant experiments and export the raw csv experiment results into results/raw-data

This step may take a few hours ... 

```
$ python3 scripts/run_experiments_parrallel.py
```

## Step 5: generate plots 

This script will generate plots from the exported data, as well as export extra data for timely plots. 

```
$ python3 scripts/run_plots.py
```

## Reset 

This script will reset the environment by
1. deleting the databases 
2. deleting raw data 
3. deleting plots 
4. deleting old message files
5. deleting old executable 

```
$ python3 scripts/run_reset.py
```

## Notes

* to run custom experiments, refer to README.md and documentation in docs directory for instructions 
* to utilize scripts to run custom p and k parameter combinations, buffer size choices, and message generation options, modify the scripts/parameters.py file 
* experiment names must be unique. to rerun an experiment, either delete the record of the old experiment from all tables, or create a new unique experiment name. 

### Re-running experiments efficiently without reimporting datasets and re-preparing data 

To rerun experiments with different configuration parameters efficiently without completely deleting the database files, follow these steps: 

1. clear DBs and results directory while preserving events, encounters, and encounter enumerations tables. 

this will clear all experiment records to avoid duplicate experiment name issues. 

```
$ python3 scripts/run_clear_dbs.py
```

this will clear the results folder (but keep the directory structure)

```
$ python3 scripts/run_clear_results.py
```

Experiment records and results directories must be cleared before rerunning experiments. 

2. modify parameters as necessary. 

to modify p and k value combinations, buffer size choices, and message generation types, see scripts/parameters.py.

other configuration parameters can be modified via cmd/cadence/configs/NDSS and pkg/logic/logic_configs/NDSS

3. rerun experiments 

```
$ python3 scripts/run_experiments_parrallel.py
```

4. generate plots 

```
$ python3 scripts/run_plots.py
```

Note: for docker, rerunning experiments with new parameters requires rebuilding the container. 

### plot reference 

plots are named based on this convention:

METRIC_K-VALUE_DATASET.pdf

METRICS:

al - message load (average number of messages in the network per time frame)
custom - message load efficiency: delivery rate / average load
custom2 - delivery efficiency: delivery rate / network load
dr - delivery rate 
lat - message delivery latency (time in days)
nl - network load (sum of all bandwidths (messages transferred in an encounter))

(plotting scripts also produce some extra plots not used in the paper)

## Docker ##

Instructions to run the simulation using docker. 

1. Make sure you are in the top-level directory of this repository.

2. Build. This will setup the environment, build the simulator, and import the datasets. 

```
docker build -t mirage-artifact .
```

3. Create a place to store results 

```
mkdir /tmp/place-to-store-results
```

4. Run the main command. Make sure you use the `-v` option to specify an accessible location for results. 

```
docker run -v /home/user/place-to-store-results:/app/results mirage-artifact
```

## Reference Papers: 

Please cite these papers when using the simulator.

```bibtex
@inproceedings{berger2023cadence,
  title={Cadence: A Simulator for Human Movement-based Communication Protocols},
  author={Berger, Harel and Sherr, Micah and Aviv, Adam},
  booktitle={Proceedings of the 16th Cyber Security Experimentation and Test Workshop},
  pages={26--31},
  year={2023}
}

@inproceedings{zacharyratliffMiragePrivateMobilitybased2026,
  title = {Mirage: {{Private}}, {{Mobility-based Routing}} for {{Censorship Evasion}}},
  booktitle = {Network and {{Distributed System Security Symposium}} ({{NDSS}})},
  author = {Ratliff, Zachary and Yang, Ruoxing (David) and Bai, Avery and Berger, Harel and Sherr, Micah and Mickens, James},
  year = {2026},
  month = feb
}