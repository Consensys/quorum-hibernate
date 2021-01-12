# Node Manager

## Introduction
In large networks it is likely that some nodes do not receive or initiate transactions for extended periods of time. These nodes incur a potentially unwanted infrastructure cost. 

Node Manager provides a solution to this problem by monitoring a node's API traffic and stopping (hibernating) the node if it has not had any API activity for a significant period of time.

## Features

* Monitors a linked Blockchain Client and Privacy Manager for inactivity.
    * Supported Blockchain Clients: **GoQuorum** and **Besu**.
    * Supported Privacy Managers: **Tessera**.
* Acts as a proxy for the Blockchain Client and Privacy Manager.
* Hibernates the linked Blockchain Client and Privacy Manager if the period of inactivity exceeds a configurable limit.
* Restarts (wakes up) the Blockchain Client and Privacy Manager when new transaction or API requests are received.
* Does not require the entire network to be using Node Managers.
* Periodically wakes up the node (configurable) to allow it to sync with the network and ensure it does not fall too far behind. 
* 1-way and 2-way (mutual) TLS supported on all of Node Manager's servers, clients, and proxies.

## Build & Run
### Pre-Requisites
    golang 1.15+
### Build
```bash
go install
```
### Run
```bash
node-manager --config path/to/config.json --verbosity 3
```

| Flag | Description |
| :---: | :--- |
| `--config` | Path to `.json` or `.toml` [configuration file](docs/config.md) |
| `--verbosity` | Logging level (`0` = `ERROR`, `1` = `WARN`, `2` = `INFO`, `3` = `DEBUG`) |

### Docker

Alternatively the [`quorumengineering/node-manager`](https://hub.docker.com/r/quorumengineering/node-manager) Docker image can be used, for example:

```bash
docker run \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -p 8081:8081 -p 9091:9091 -p 9391:9391 \
    --mount type=bind,source=/path/to/nm.json,target=/config.json --mount type=bind,source=/path/to/peers.json,target=/peers.json \
    quorumengineering/node-manager:latest -config /config.json
```

*Note: `-v /var/run/docker.sock:/var/run/docker.sock` allows the Node Manager container to start/stop Blockchain Client/Privacy Manager containers.*

## Configuration
See [docs/config.md](docs/config.md) for a full description of all configuration options.

## Deployment/Usage
See [docs/deployment.md](docs/deployment.md) for details on adding and using Node Manager in networks. 

## How It Works
See [docs/how-it-works.md](docs/how-it-works.md) for an overview of the processes used by Node Manager and common errors.

## Sample Configurations
See [docs/samples](docs/samples) for sample configuration files for various network types.


