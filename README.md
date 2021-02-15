![logo](logo.png)

- [Node Hibernate](#node-hibernate)
  - [Introduction](#introduction)
  - [Features](#features)
  - [Build and Run](#build-and-run)
    - [Pre-Requisites](#pre-requisites)
    - [Build](#build)
    - [Run](#run)
    - [Docker](#docker)
  - [Configuration](#configuration)
  - [Deployment/Usage](#deploymentusage)
  - [Architecture](#architecture)
  - [Sample Configurations](#sample-configurations)
  - [Demo](#demo)

# Quorum Hibernate

## Introduction

In large networks it is likely that some nodes do not receive or initiate transactions for extended periods of time. These nodes incur a potentially unwanted infrastructure cost. 

Node Hibernate provides a solution to this problem by monitoring a node's API traffic and stopping (hibernating) the node if it has not had any API activity for a significant period of time.

## Features

* Monitors a linked Ethereum Client and Privacy Manager for inactivity.
    * Supported Ethereum Clients: **GoQuorum** and **Besu**.
    * Supported Privacy Managers: **Tessera**.
    * Supported consensus
        * GoQuorum: Istanbul BFT, Raft and Clique
        * Besu: Clique
* Acts as a proxy for the Ethereum Client and Privacy Manager.
* Hibernates the linked Ethereum Client and Privacy Manager if the period of inactivity exceeds a configurable limit.
* Restarts (wakes up) the Ethereum Client and Privacy Manager when new transaction or API requests are received.
* Does not require the entire network to be using Node Managers.
* Periodically wakes up the node (configurable) to allow it to sync with the network and ensure it does not fall too far behind. 
* 1-way and 2-way (mutual) TLS supported on all of Node Hibernator's servers, clients, and proxies.

## Build and Run
### Pre-Requisites
    golang 1.15+
### Build
```bash
go install
```
### Run
```bash
node-hibernator --config path/to/config.json --verbosity 3
```

| Flag | Description |
| :---: | :--- |
| `--config` | Path to `.json` or `.toml` [configuration file](docs/config.md) |
| `--verbosity` | Logging level (`0` = `ERROR`, `1` = `WARN`, `2` = `INFO`, `3` = `DEBUG`) |

### Docker

Alternatively the [`quorumengineering/node-hibernator`](https://hub.docker.com/r/quorumengineering/node-hibernator) Docker image can be used, for example:

```bash
docker run \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -p 8081:8081 -p 9091:9091 -p 9391:9391 \
    --mount type=bind,source=/path/to/nh.json,target=/config.json --mount type=bind,source=/path/to/peers.json,target=/peers.json \
    quorumengineering/node-hibernator:latest -config /config.json
```

*Note: `-v /var/run/docker.sock:/var/run/docker.sock` allows the Node Hibernator container to start/stop Ethereum Client/Privacy Manager containers.*

## Configuration
See [docs/config.md](docs/config.md) for a full description of all configuration options.

## Deployment/Usage
See [docs/deployment.md](docs/deployment.md) for details on adding and using Node Hibernator in networks. 

## Architecture
See [docs/architecture.md](docs/architecture.md) for an overview of the processes used by Node Hibernator and common errors.

## Sample Configurations
See [docs/samples](docs/samples) for sample configuration files for various network types.

## Demo
See [demo](demo) for a Docker Compose demo network that can be used for initial experimentation with Node Hibernator. 

