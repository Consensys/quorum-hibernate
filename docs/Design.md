# Node Manager: Design & Processes

## What does Node Manager do?

![Architecture & Design](images/node-manager-arch.jpg)

The above diagram depicts a simple 3 node privacy-enabled network where each node uses Node Manager.

Each Node Manager:

* Acts as a proxy for its linked Blockchain Client and Privacy Manager.  As all incoming traffic goes through Node Manager, it is able to monitor the activity on the linked nodes. 
  
* Communicates with other Node Managers to retrieve the statuses of their linked nodes.

## Process: Hibernation of node after inactivity

![node hibernation flow](images/node-hibernation-flow.jpg)

The above sequence diagram outlines the process of hibernating a node after the configured inactivity period has been reached.  In more detail:

* **1.1:** Node Manager monitors the incoming requests for the linked Blockchain Client and Privacy Manager to determine whether the node is active.  The inactivity timer is reset on incoming proxy requests.

* **1.2:** If the node has been inactive for more than the configured period, Node Manager initiates the pre-hibernation process.

* **1.2.1 to 1.2.3:** Node Manager checks if the node is safe to hibernate by fetching network and consensus information from the Blockchain Client. These checks ensure the network will continue to be operational if the node hibernates. See [Consensus Checks](#Consensus-Checks) for further details.
  
  * If the validation is successful, Node Manager proceeds with hibernating the node.
  * If the validation is unsuccessful, Node Manager aborts the hibernation process, resets the inactivity timer and waits for the next inactivity trigger to attempt hibernation again.

* **1.3 to 1.3.3:** Node Manager checks if the node is safe to hibernate by reaching out to the other Node Managers in the network to see if any have also initiated hibernation of their node.  This check ensures that multiple Node Managers do not perform hibernation at the same time which could break the consensus checks performed earlier.
  
  * If no other Node Managers have initiated hibernation, Node Manager proceeds with hibernating the node.
  * If another Node Manager has initiated hibernation or did not respond, Node Manager aborts the hibernation process, resets the inactivity timer and waits for the next inactivity trigger to attempt hibernation again.

* **1.4:** Node Manager hibernates the local Blockchain Client and Privacy Manager.

### Consensus Checks

Node Manager ensures that hibernation would not result in a break in consensus.  The checks performed depend on the consensus mechanism of the network and whether Node Manager was started in `strict` mode (default) or if `disableStrictMode` was set to `true`.

| Consensus Engine | Strict Mode | Normal Mode |
| :---: | :--- | :--- |
| Raft (GoQuorum) | - **Minter** and **Peer** nodes cannot be hibernated. <br /> <br /> - **Learner** nodes can be hibernated | - **Minter** nodes cannot be hibernated <br /> <br /> - Up to ***49%*** of **Peer** nodes can be hibernated <br /> <br />- **Learner** nodes can be hibernated
| Istanbul (GoQuorum) | - **Validator** nodes cannot be hibernated <br /> <br /> - **Non-Validator** nodes can be hibernated | - Up to ***f*** **Validator** nodes can be hibernated (in a network with ***3f + 1*** Validator nodes) <br /> <br /> - **Non-Validator** nodes can be hibernated
| Clique (GoQuorum) | - **Signer** nodes cannot be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated | - Up to ***49%*** of **Signer** nodes can be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated
| Clique (Besu) | - **Signer** nodes cannot be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated | - Up to ***49%*** of **Signer** nodes can be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated

## Process: Waking of node after new activity

The below diagram depicts a private transaction flow for the following scenario:
* Node `A` and Node `B` are both running a GoQuorum and Tessera nodes with individual Node Managers monitoring each node
* Because of inactivity both GoQuorum and Tessera nodes are down
* Node Manager `A` receives a private transaction request 

![request flow](images/node-manager-flow.jpg)

The flow is as described below:

* **1.0:** A new RPC request for private transaction between GoQuorum Node `A` and `B` is sent to the Node Manager `A` (acting as a proxy for GoQuorum `A`). Node Manager `A` parses the transaction requests. For private transactions it extracts the privacy manager public keys from the `privateFor` argument of request. It then checks if there are entries in  [peers config](./CONFIG.md#Peers-config-file) for the public keys and identifies the remote Node Managers. If there are no entries, it assumes that the node is not managed by a Node Manager.

*  **1.1:** Node Manager `A` initiates the process to check if the local GoQuorum node and all recipient Tessera (Node `B`'s Tessera in this case) are up. 

* **1.2:** Node Manager `A` checks the status of local GoQuorum and Tessera. If these are down, triggers restart.

* **1.3.1 to 1.3.4:** Node Manager `A` sends a request to Node Manager `B` via RPC call to check the remote node status. Node Manager `B` checks the status of linked GoQuorum and Tessera nodes. If the nodes are down it initiates the restart and responds back with status.

* **1.4:** Once all nodes are up, Node Manager `A` forwards the request to Node `A`'s GoQuorum for processing.

* **1.4.1 to 1.4.8:** This is standard private transaction processing flow of GoQuorum. Once the private transaction is processed, GoQuorum responds back to Node Manager `A` with response.

* **1.4.9, 1.4.10:** Node Manager `A` receives the response for the  transaction and responds back to client

## Error handling for user
User requests to Node Manager will fail under the following scenarios.

| Scenario  | Error message received by user | Action required |
| --- | --- | --- |
| Node Manager receives a request from user while block chain client and privacy manager are being stopped by it due to inactivity. | 500 (Internal Server Error) - `node is being shutdown, try after sometime` | Retry after some time. |  
| Node Manager receives a request from user while block chain client and privacy manager are being started up by it due to activity. | 500 (Internal Server Error) - `node is being started, try after sometime` | Retry after some time. |  
| Node Manager receives a private transaction request from user and participant node(of the transaction) managed by Node Manager is down. | 500 (Internal Server Error) - `Some participant nodes are down` | Retry after some time. |  
| Node Manager receives a request from user when starting/stopping of block chain client or privacy manager by Node Manager failed. | 500 (Internal Server Error) - `node is not ready to accept request` | Investigate the cause of failure and fix the issue. |  

Node Manager will consider its peer is down and proceed with processing if it is not able to get a response from its peer Node Manager when it tries to check the status for stopping nodes or handling private transaction.
