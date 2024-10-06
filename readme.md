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

- run the node  
`-bootstrap` init cluster for first time  
`-addr` bind address for this node to listen traffic 
``` shell 

# practically node will deploy in different machine or host so it can use same default port.
# for the example it will run in same host but with different port
# to see detail of cmd use '-h'

# start cluster
# start node 1 (bootstrap)
./cmd/scarlettd -bootstrap -name=node-1 -addr=localhost:8001
```  

- add node to cluster  
`-join` remote (raft-node) address that will receive join request  
``` shell
# start node 2, the value of 'join' is remote address of node 1 that will receive join request from this node 
./cmd/scarlettd -name=node-2 -addr=localhost:8002 -join=localhost:8001

# start node 3
./cmd/scarlettd -name=node-3 -addr=localhost:8003 -join=localhost:8001
```

#### HTTP
it has two endpoint, `POST /command` for mutate the data and `GET /query` for retrival,
command can sent to any node in same cluster and it automatically find the leader to redirect the request.

send request to command endpoint to set data (state).
``` 
curl --request POST --url http://localhost:8001/command --data '{"cmd":"set","key":"my-awesome-key","value":"my-awesome-value"}'
```

send request to query endpoint for data retreival.
``` 
curl --request GET --url http://localhost:8001/query --data '{"cmd":"get","key":"my-awesome-key"}'
```

#### Todo:
* TLS
* Authentication
* Cluster Management
