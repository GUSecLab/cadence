# Configuration files

The following config files are utilized in Cadence:

1. General config files
2. Logic config files

Each example is acompanied by slight comments on the specific field.
The config files use JSON format.

## General conf files
Example file:

```
{
  "top_level": {
    "log": "DEBUG",
    "db": "sqlite",
    "dbfile": "sqlite_test.db",
    "time_format": "2006-01-02 15:04:05.000" ,
    "seed": 12345 
  },
  "web_server": {
    "port": 8082,
    "host": "localhost"
  },
  "simulation": {
    "start_time": 1201959058,
    "end_time": 1202492359,
    "dataset_name": "tdrive",
    "experiment_name": "tdrive - mirage run=3 p=0.9 k=6",
    "conditions_file": "conditions.json",
    "logic": "mirage", 
    "logic_file": "../../pkg/logic/logic_configs/mirage/logic_config_tdrive_mirage.json",
    "messages_file": "../../../marathon-files/messages_tdrive_dm_x10.json",
    "messages_template": "messages_template.json",
    "generator_script": "message_generator.py",
    "message_generation_type":2, 
    "min_buffer_size": 100,
    "max_buffer_size": 100,
    "min_message_size": 1.0,
    "max_message_size": 1.0,
    "time_step": 60.0,
    "encounter_isolator": 60.0,
    "N_encounters_split": 2
  },
  "cli":{
    "lens": "tdrive",
    "path": "../../marathon-mobility-data/tdrive-small",
    "name": "tdrive"
  }
}


```

## Logic conf files
Example file:
```
{
    "logic":"mirage",
    "dataset_name": "tdrive",
    "underlyning_logic": "mirage",
    "profile_file": "profile_defs.json",
    "organizer": 1,
    "districts_amount": 100,
    "p": 0.9,
    "epsilon": 0.1,
    "k": 6,
    "kmeans_iterations": 50,
    "max_n": 100,
    "n_factor": 3,
    "noise_sigma": 0.1 
} 
```

## Condtions conf file
```
{
  "conditions" : [
    {
      "name" : "distance200m", //maximum distance between nodes
      "type" : "distance",
      "params" : {
        "dist" : 200.0
      }
    },
    {
      "name" : "probability50", //probability of encounter
      "type" : "probability",
      "params" : {
        "prob" : 1.0
      }
    },
    {
      "name" : "transition", //minimum time between encounters
      "type" : "whenhaveisawu",
      "params" : {
        "transition_time" : 600000.0
      }
    },
    {
      "name" : "updates", // updates' based encounters
      "type" : "doYouHaveEnoughUpdates"
    }
  ]
}
```

## Attackers config file
Example file:
```
[
  {
    "type" : "dropper", // dropper type
    "amount": 0.1 // ratio of nodes that match this type
  },
  {
    "type" : "flooder",
    "amount": 0.1
  }
]
```