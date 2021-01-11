# Node Manager

### Introduction
In large networks it is possible that some nodes in the network have low transaction volumes and probably do not receive or initiate transactions for days. However, the node keeps running incurring the infrastructure cost. One of the requirements has been to proactively monitor the transaction traffic at a node and stop the node if its inactive for long.

Node Manager is designed to cater to above requirement. The tool is built to:

* Monitor a linked Blockchain Client and Privacy Manager for inactivity
* Hibernate the linked Blockchain Client and Privacy Manager if its inactive beyond certain configured time
* Restart the Blockchain Client and Privacy Manager upon new transaction/calls 

Node Manager acts as a proxy for the Blockchain Client and Privacy Manager nodes. When running with Node Manager it is expected that all clients would submit requests to the corresponding Node Manager proxy servers instead of directly to the Blockchain Client or Privacy Manager nodes.

### Key Features

- Node Manager supports both **pure** and **hybrid** deployment models. In a pure deployment model, all nodes have a Node Manager instance running. However, in hybrid deployment model, it is possible to have few nodes with Node Manager running and few without Node Manager.  
- **Periodic sync** feature allows nodes to be brought up periodically to ensure that its synced with the network. 
- **TLS**: 1-way and 2-way (mutual) TLS can be configured on each of Node Manager's servers, clients, and proxies.  
- Currently supports: 
    - **GoQuorum** and **Besu** block chain clients
    - **Tessera** as Privacy Manager

### Build & Run

```bash
node-manager --config path/to/config.json --verbosity 3
```

| Flag | Description |
| :---: | :--- |
| `--config` | Path to `.json` or `.toml` [configuration file](docs/Config.md) |
| `--verbosity` | Logging level (`0` = `ERROR`, `1` = `WARN`, `2` = `INFO`, `3` = `DEBUG`) |

Alternatively the [`quorumengineering/node-manager`](https://hub.docker.com/r/quorumengineering/node-manager) Docker image can be used, for example:

```bash
docker run \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -p 8081:8081 -p 9091:9091 -p 9391:9391 \
    --mount type=bind,source=/path/to/nm.json,target=/config.json --mount type=bind,source=/path/to/peers.json,target=/peers.json \
    quorumengineering/node-manager:latest -config /config.json
```

*Note: `-v /var/run/docker.sock:/var/run/docker.sock` allows the Node Manager container to start/stop Blockchain Client/Privacy Manager containers.*

### Design
Refer [here](docs/Design.md) for Node Manager design and flows.

### Configuration
Refer [here](docs/Config.md) for configuration details.

### Examples
Refer [here](examples/README.md) for sample configuration files and start scripts


