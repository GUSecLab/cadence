## LOGICS

a brief overview of each logic engine. 

### Overview

For all logic engines, a message is only passed during an encounter if the following minimum conditions are met:

1. The receiver node does not possess the message. 

2. The message is alive, defined as: 

Time-To-Live Hops (TTLHops) is greater than 0, indicating that the amount of transfers the message has experienced has not exceeded its survival limit.

Time-To-Live Time (TTLTime) is greater than 0, indicating that the lifetime of the message (amount of time the message is allowed to survive for) has not passed.

**`organizer` sets the buffer type. 1 indicates bounded buffer while 2 indicates unbounded infinite buffer. Bounded buffer eviction is handled on a FIFO basis**

### Broadcast

Passes all messages between all nodes belonging to an encounter. 

### Randomwalk V1

Each message is passed between nodes with a specified probability. 

use `SetTransferProbability(probability)` to specify transfer probability. For example, if probability == 0.8, aproximately 80% of messages will be passed from the sender to the receiver node. 

### Randomwalk V2

Each message is passed between nodes based on a time constant probability check. The time constant describes a relationship between the encounter time and the message creation time. The difference between encounter time and message creation time is known as time_held, representing the amount of time a message has existed since its creation. 

Randomwalk V2 logic describes a situation where "younger" messages, messages with a lower time_held value, have a lower chance of getting passed from the sender to the receiver node. There are three possible scenarios:

1. The message was created at the time of or after the time of encounter (the message has 0 or negative time_held value).
- the message will not be passed.

2. The message's time_held value is greater than the specified time constant
- the message will be passed with 100% probability.

3. The message's time_value is lesser than the specified time constant
- the message will be passed with probability of time_held/timeConstant

Example:

Message Creation Time: 1800
Encounter Time: 2000
Time Constant: 1000

Time Held = 2000 - 1800 = 200

Transfer Probability = 200 / 1000 = 0.2 = 20%

approximately 20% of messages will be transferred in this scenario

use `SetConstant(constant)` to specify time constant

### Addressing (PPBR)

"addressing" implements the probabilistic profile-based routing (PPBR) protocol. 

parameters:

`
{
    "logic":"addressing",
    "dataset_name": "tdrive",
    "underlyning_logic": "addressing",
    "profile_file": "profile_defs.json",
    "organizer": 1,
} 

`

profile_defs.json

`
  {
    "gridsize" : 200,
    "distance": 20
  }
`

1. PPBR logic takes a profile definition file, which defines parameters for the PPBR district grid. 

### mirage (MIRAGE)

"mirage" implements the MIRAGE routing protocol.

parameters:

`
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
    "noise_sigma": 0.1,
    "attack_type": "none"
} 
`

1. districts_amount specifies the number of districts used for location profiling. 

2. p is a DP parameter. 1-p is the probability of flipping any particular edge in a mobility graph. 

3. epsilon is deprecated 

4. k is the k-top frequently visited districts used in mobility graph generation. 

5. kmeans_iterations is the amount of iterations used to generate profiling districts. 

6. max_n is the upper bound for n values in the n value map that dictates message passing attempts. 

7. n_factor is the coefficient used to scale n values in the n value map for practicality. 

8. noise_sigma specifies the magnitude (standard deviation) for the gaussian noise added to the global mobility graph. This is only relevant if attack type is noise. 

9. attack_type specifies the type of attack performed. options are "noise", "sybil", and "none".