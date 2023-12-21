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
    "log": "INFO", //log level
    "db": "mysql", //name of database
    "dbfile": "USERNAME:PASSWORD@tcp(127.0.0.1:3306)/SCHEMA?charset=utf8mb4&parseTime=True&loc=Local", //string of connection to the db, based on username:password, IP address, schema name, and other parameters 
    "time_format": "2006-01-02 15:04:05.000" 
    "seed": 12345 
  },
  "web_server": { //configurations for graphical web service
    "port": 8080,
    "host": "localhost"
  },
  "simulation": {
    "start_time": 1212498842,
    "end_time": 1212505535,
    "dataset_name": "hand",
    "experiment_name": "broadcast_hand2",
    "conditions_file": "conditions.json", //path to conditions file
    "logic": "broadcast", 
    "logic_file": "../../pkg/logic/logic_config_hand.json", //path to logic configuration file
    "messages_file": "../../../marathon-files/messages_hand.json", //path to messages file
    "messages_template": "messages_template.json", //path to messages template file
    "generator_script": "message_generator.py", //path to message generator file
    "message_generation_type": 1, //message generation type - random pairs (0) or N^2 messages
    "min_buffer_size": 1000, //buffer min/max
    "max_buffer_size": 1000,
    "min_connection_duration": 1, //min/max connection times
    "max_connection_duration": 2,
    "min_message_size": 1.0, // min/max message size
    "max_message_size": 1.0,
    "min_events_per_node": 0, //min amount of events to be included in the message creation
    "time_step": 100.0, // timestep for discrete split of the data
    "encounter_isolator": 100.0, // isolation between timesteps - not implemented yet
    "N_encounters_split": 2 // splitting big encounters by N encounters, if desired
  },
  "cli":{
    "lens": "napa",
    "path": "../../marathon-mobility-data/napa",
    "name": "napa"
  }
}
```

## Logic conf files
Example file:
```
{
    "logic": "mixed", 
    "dataset_name": "geolife",
    "underlyning_logic": "mixed", //underlyning logic in case of attackers
    "profile_file": "profile_defs.json", //profile for HumaNets protocol 
    "randomwalk_transfer_probability": 0.5, //P probability for randomwalk-v1
    "randomwalk_delete_probability": 1, //Q probability for randomwalks
    "randomwalk_transfer_constant": 1, //Constant values for randomwalk-v2
    "mixed_randomwalk_version": 1, //randomwalk version for mixed
    "mixed_ideality_version": 0, //mixed ideality algo
    "ideality_coin_flip": 0.7, //probability for mixed algo coin flip
    "ideality_distance_n": 1, // N last nodes for mixed algo
    "organizer": 2 //organizer governs if the buffers are bounded or not, 1 for bounded, 2 for unbounded
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
