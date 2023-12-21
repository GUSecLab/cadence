# Cadance messages.json file format

Example:

```
[
  {
    "id" : 1,
    "sender" : 55,
    "destination" : 56,
    "type" : 10,
    "time": 1206349104
  },
  {
    "id" : 2,
    "sender" : 92,
    "destination" : 55,
    "type" : 10,
    "time" : 1200003170
  }
]
```

Cadence simulations typically include message passing.  To indicate where and when messages originate, a messages.json file must be specified.  (This file can be named whatever you like, and is passed into Cadence using the `-m` or `--messages` option.)

As shown above, the JSON file is an array of dictionaries, where each (dictionary) entry specifies a message.  The above example has two messages.

Each message has a unique identifier ("id").  These need not be consecutive, but they must be integers and they must be unique.

The "sender" specifies the NodeID of the originator of the message.  Similarly, "destination" indicates the intended recipient(s) of the message.  In the case of unicast messages, this should be a single NodeID.  If multicast, it should be an array of NodeIDs.

The "time" field indicates when the originator effectively hits the "send" button on their device.  Note that the first transmission of the message may occur after this time, if no encounter takes place exactly at "time".

The "type" field indicates the type of message (e.g., unicast, multicast, broadcast, or anycast).  Current types are defined in [/pkg/logic/message.go](/pkg/logic/message.go).  Note that the "type" field must correspond to the integer value of the address type.

