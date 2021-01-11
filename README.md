# Node Manager

### Introduction
In large networks it is possible that some of the nodes in the network have low transaction volumes and probably do not receive or initiate transactions for days. However, the node keeps running incurring the infrastructure cost. One of the requirements has been to proactively monitor the transaction traffic at a node and stop the node if its inactive for long.

Node manager is designed to cater to above requirement. The tool is built to:

* Monitor a linked blockchain client and privacy manager for inactivity
* Hibernate the linked blockchain client and privacy manager if its inactive beyond certain configured time
* Restart the blockchain client and privacy manager upon new transaction/calls 

Node Manager acts as a proxy for the blockchain client and privacy manager nodes. When running with node manager it is expected that all clients would submit requests to the corresponding node manager proxy servers instead of directly to the blockchain client or privacy manager nodes.

### Key Features

- Node manager supports both **pure** and **hybrid** deployment models. In a pure deployment model, all nodes have a node manager instance running. However in hybrid deployemnt model, it is possible to have few node with node manager running and few without  
- **Periodic sync** feature allows nodes to be brought up periodically to ensure that its synced with the network. 
- **TLS**: 1-way and 2-way (mutual) TLS can be configured on each of Node Manager's servers, clients, and proxies.  
- Currently supports: 
    - **GoQuorum** and **Besu** block chain clients
    - **Tessera** as privacy manager

### Design
Refer [here](docs/Design.md) for node manager design and flows.

### Build, Run and Configuration
Refer [here](CONFIG.md) for build, run & configuration details.


