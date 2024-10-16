#### Why scarlett?
scarlett is distributed key-value store uses a consensus protocol (raft).



#### Feature
- http-api
- easy to deploy
- Reliable: Fully replicated key-value store provides fault-tolerance and high-availability.




### Quickstart
build from source 
``` shell
# clone the project
git clone https://github.com/odit-bit/scarlett
cd ./scarlett

# build exe file
go build ./cmd/scarlettd
```
or use prebuilt execute file from `./bin/scarlettd`
- run the node  
``` shell 

# practically node will deploy in different machine or host so it can use same default port.
# for the example it will run in same host but with different port
# to see detail of cmd use '-h'

# start cluster
# start node 1 (bootstrap)
./bin/scarlettd -name=node-1 -raft=8001 -http=9001 -bootstrap -raft-dir=./
```  

- add node to cluster  
`-join` remote (raft-node) address that will receive join request  
``` shell
# start node 2, the value of 'join' is remote address of node 1 that will receive join request from this node 
./bin/scarlettd -name=node-2 -raft=8002 -http=9001 -join=localhost:8001 -raft-dir=./

# start node 3
./bin/scarlettd -name=node-3 -raft=8003 -http=9003 -join=localhost:8001 -raft-dir=./
```

#### HTTP
it has two endpoint, `POST /command` for mutate the data and `GET /query` for retrival,
command can sent only to leader node (for now) usually the node that bootstraped for first time, while query can sent to any node in same cluster

send request to command endpoint to set data (state).
``` 
curl --request POST --url http://localhost:9001/command --data '{"cmd":"set","key":"my-awesome-key","value":"my-awesome-value"}'
```

send request to query endpoint for data retreival.
``` 
curl --request GET --url http://localhost:9001/query --data '{"cmd":"get","key":"my-awesome-key"}'
```

send request to endpoint for delete (state).
``` 
curl --request POST --url http://localhost:9001/command --data '{"cmd":"delete","key":"my-awesome-key"}'
```


#### Todo:
* TLS
* Authentication
* Cluster Management
* Observability
