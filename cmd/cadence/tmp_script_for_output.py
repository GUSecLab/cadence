import mysql.connector,sys
import json
import os
import random
#make sure the dir exists
def create_directory_if_not_exists(file_path):
    directory = os.path.dirname(file_path)
    if not os.path.exists(directory):
        os.makedirs(directory)
        print(f"Directory '{directory}' created.")
#generate message 1-1
def generate_messages_11(filename, nodes_and_times, min_message_size, max_message_size):
    with open(filename) as file:
        messages = json.load(file)
    #if we only have one template, change it to array
    if type(messages)==type({}):
        mes=[]
        mes.append(messages)
        messages=mes
    res = {}
    #create pairing
    pairs = {}
    available_numbers = nodes_and_times.copy()

    for num in nodes_and_times:
        pair = random.choice(available_numbers)
        pairs[num[0]] = pair[0]
        pairs[pair[0]] = num[0]
        available_numbers.remove(pair)
    #odd amount of nodes
    if len(available_numbers)!=0:
        pairs[available_numbers[0][0]]=nodes_and_times[0]
    
    id_count = 0

    for m in messages:
        m['lathops'] = 0
        for t in nodes_and_times:
            node=t[0]
            m_tmp = m.copy()
            m_tmp['sender'] = node
            m_tmp['time']=t[1]
            m_tmp['size'] = min_message_size + random.random() * (max_message_size - min_message_size)
            rand_node=pairs[node]
            m_cur = m_tmp.copy()
            m_cur['id'] = str(id_count)
            id_count += 1
            if m_cur['shards'] == 1:
                m_cur['id'] += "_" + str(m_cur['shardid'])
            m_cur['destination'] = rand_node
            m_cur['destinationnode'] = rand_node
            if node not in res:
                res[node] = []
            res[node].append(m_cur)

    return res
#generate messages to everyone
def generate_messages_N2(filename, nodes_and_times, min_message_size, max_message_size):
    with open(filename) as file:
        messages = json.load(file)
    #if we only have one template, change it to array
    if type(messages)==type({}):
        mes=[]
        mes.append(messages)
        messages=mes
    res = {}

    id_count = 0

    for m in messages:
        m['lathops'] = 0
        for t in nodes_and_times:
            node=t[0]
            m_tmp = m.copy()
            m_tmp['sender'] = node
            m_tmp['size'] = min_message_size + random.random() * (max_message_size - min_message_size)
            timer=0
            #pick from the pairing
            for rand_node_data in nodes_and_times:
                rand_node=rand_node_data[0]
                if rand_node == node:
                    continue
                m_cur = m_tmp.copy()
                m_cur['time']=t[1]+timer
                timer+=10
                m_cur['id'] = str(id_count)
                id_count += 1
                if m_cur['shards'] == 1:
                    m_cur['id'] += "_" + str(m_cur['shardid'])
                m_cur['destination'] = rand_node
                m_cur['destinationnode'] = rand_node
                if node not in res:
                    res[node] = []
                res[node].append(m_cur)

    return res

#parameters
#1-db string
#2-dataset name
#3-template file
#4-min message size
#5-max message size
#6-start time of experiment
#7-output file
#8- N^2 messages or 1-1 messages
#9-end time
#10-minimum distance between nodes

# Get MySQL connection string from user input
connection_string = sys.argv[1]

# Create MySQL connection
connection = mysql.connector.connect(
    user=connection_string.split(":")[0],
    password=connection_string.split(":")[1].split("@")[0],
    host=connection_string.split("@")[1].split("/")[0].split("(")[1].split(":")[0],
    port=int(connection_string.split("@")[1].split("/")[0].split("(")[1].split(":")[1].split(")")[0]),
    database=connection_string.split("/")[1].split("?")[0],
    charset='utf8mb4',
    use_pure=True
)

# Check if the connection was successful
if connection.is_connected():
    print('Connected to MySQL!')

# Execute the query
# dataset_name = sys.argv[2]  # dataset name
# start_time=sys.argv[6]
# end_time=sys.argv[9]
# distance=sys.argv[10]
cursor = connection.cursor()
query = """
    SELECT t.*
FROM encounters t
JOIN (
    SELECT node_id
    FROM (
        SELECT node1 AS node_id, COUNT(*) AS node_count
        FROM encounters
        GROUP BY node1
        UNION ALL
        SELECT node2 AS node_id, COUNT(*) AS node_count
        FROM encounters
        GROUP BY node2
    ) AS node_counts
    ORDER BY node_count DESC
    LIMIT 64
) AS popular_nodes
ON t.node1 = popular_nodes.node_id OR t.node2 = popular_nodes.node_id;
 
"""
params = []#,start_time,end_time,distance)# float(sys.argv[6]))  # Replace `startTime` with the appropriate value
cursor.execute(query, params)
results=cursor.fetchall()
data=["dataset_name,distance,experiment_name,time,node1,node2,duration,x,y,z\n"]
for r in results:
    data.append(",".join(map(str,r))+"\n")
with open("/home/harel/results.csv","w") as w:
    for d in data:
        w.write(d)
exit(0)

nodes_times={}
nodes_oc={}
for i in results:
    node1=i[0]
    node2=i[1]
    t=float(i[2])
    if node1 in nodes_times:
        nodes_times[node1]=min(t,nodes_times[node1])
        nodes_oc[node1]+=1
    else:
         nodes_times[node1]=t
         nodes_oc[node1]=1
    if node2 in nodes_times:
        nodes_times[node2]=min(t,nodes_times[node2])
        nodes_oc[node2]+=1
    else:
         nodes_times[node2]=t
         nodes_oc[node2]=1
enough=[x for x in nodes_oc if nodes_oc[x]>10]
results_short=[]
for x in nodes_times.keys():
    if x in enough:
        results_short.append([x,nodes_times[x]])
# results=results_short

# Close the cursor and connection
cursor.close()
connection.close()

#create the messages and write them to file
messages=[]
if sys.argv[8]=="0":   
    messages=generate_messages_11(sys.argv[3], results, float(sys.argv[4]), float(sys.argv[5]))
else:
    messages=generate_messages_N2(sys.argv[3], results, float(sys.argv[4]), float(sys.argv[5]))

#making sure that the folder exists
create_directory_if_not_exists(sys.argv[7])
#dump the file
with open(sys.argv[7], "w") as json_file:
    json.dump(messages, json_file)

print("messages dumped to", sys.argv[7])


