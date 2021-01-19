# Node Manager demo

Creates a 5 node (*GoQuorum*) privacy-enabled (*Tessera*) raft network using Docker Compose.  Each node is managed by Node Manager.

This network provides an easy way to get a Node Manager network running for initial experimentation.  Nodes 4 & 5 are candidates for hibernation after periods of inactivity as they are configured with `disableStrictMode: true`.  If by coincidence Node 4 or 5 is the minter then it will not be able to hibernate.

Network customisation (e.g. different consensus mechanism, different network size, different Blockchain Clients) is not supported.  This is to keep the compose file concise and hopefully easy to understand.  It should be used as a starting point for configuring and running your own networks.

#### Usage
``` shell
docker-compose up -d
# wait for all nodes to completely start (check logs), Node Managers start once GoQuorums and Tesseras have started

# to stop
docker-compose down
```

#### Exported ports
* `5300[1-5]` - Node Manager 1-5 RPC server port
* `5310[1-5]` - Node Manager 1-5 GoQuorum proxy server port
* `5320[1-5]` - Node Manager 1-5 Tessera proxy server port