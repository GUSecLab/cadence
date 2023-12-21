# Simulator Outline


## Main Algorithm

```
main(){

   dataset = load_data_set(data_file) # load the dataset from a file or group of files
   # dataset = gen_data_set(dataset, AI/ML-Alg) 
   
   trace = make_contact_trace(dataset,       # dataset will be an input dataset, and closeness_f is pointer 
                              closeness_f,   # to a function to determine two nodes are close
                              linearization) # fill in missing data points for a node that may suggest additional btw. 
                                             # moment in the input dataset

   messages = opt.message_settings # some setttings on messages? size, number, source, dest? produce a list of message objs
   
   algorithms = opt.alg_settings # some settts on algorithms to produce a list of algs objects
   
   exprt = simulate(trace, algorithms, messages) #run all algorithms in parallel over the trace for the set of messages
   
   return exprt #export trace to be used by the mobile
}   

simulate(trace, algorithms, messages){
   for time_t in trace:
       for n_i in time_t.nodes: # list of obect nodes
           for n_j in n_i.contacts:
              for alg in algorithms:
                 (go) handoff(message, n_i,n_j, alg) 
                 
}

```

## Objects

```
object dataset:
   input_f: file input
   
```

```
object trace:
  time_0:
      n_i: {n} # nodes that n_i contacts at time_0
      n_{i+1}: {n} # nodes taht n_{i+1} contacts at time_0
  time_1:
      ...
```

```
object node:
   ident # identifier for this node
   msg_storeage[alg] #object defining the storage capacity of the node per message
   time_loc #location of the node at the time
   rand_seed[alg] #rand seed of it's rand_num_gen
   rand_num_gen[alg] #its own rng for consitency
   state[alg] # a key/value storage for whatever might need to be stored in a node for any alg.
```

```
object msg_storage:
   buffer: {messages}
   evict_strat #function point for evict messages when capacity is reached
   capacity #max size   
   get_message_by_prop_of_message # retrieve a message based on some property of a message
```
