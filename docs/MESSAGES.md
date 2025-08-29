# Cadance messages.json file format

Example:

```

  {
    "id" : "",
    "sender" : -1,
    "destination" : -1,
    "type" : 10,
    "time": 0,
    "payload": "Hi ",
    "shards": 0,
    "shardid": 0,
    "ms": false,
    "ttl_hops": 20,
    "ttl_secs": 1000000,
    "size": 1
}

```

Cadence simulations typically include message passing.  To indicate where and when messages originate, a messages.json file must be specified. 

As shown above, the JSON file is an array of dictionaries, where each (dictionary) entry specifies a message.  The above example has two messages.

Each message has a unique identifier ("id").  These need not be consecutive, but they must be integers and they must be unique.

The "sender" specifies the NodeID of the originator of the message.  Similarly, "destination" indicates the intended recipient(s) of the message.  In the case of unicast messages, this should be a single NodeID.  If multicast, it should be an array of NodeIDs.

The "time" field indicates when the originator effectively hits the "send" button on their device.  Note that the first transmission of the message may occur after this time, if no encounter takes place exactly at "time".

The "type" field indicates the type of message (e.g., unicast, multicast, broadcast, or anycast).  Current types are defined in [/pkg/logic/message.go](/pkg/logic/message.go).  Note that the "type" field must correspond to the integer value of the address type.

The "shards","shardid","ms" relate to the options of splitting messages (under construction).

The ttl values (secs and hops) refer to the amount of time or nodes a message can be transfered until it can no longer be passed.

# Message Generation 

There are a few message generation options: 

0: Direct Message - each node generates a message addressed to one other node. 

1: Group Message - messages are addressed to groups. 

2: Bulk DM - each node generates 10 messages addressed to 10 random unique nodes. 

3: Bulk DM Gradual Generation - Same logic as option 2, but messages are generated gradually throughout the span of the simulation. 

4: Bulk DM Gradual Generation Half Timespan - Similar to option 3, but messages are generated gradually throughout the first half of the simulation timespan. 