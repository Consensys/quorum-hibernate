# Node Hibernator: Design & Processes

## What does Node Hibernator do?

![Architecture & Design](images/node-hibernator-arch.jpg)

The above diagram depicts a simple 3 node privacy-enabled network where each node uses Node Hibernator.

Each Node Hibernator:

* Acts as a proxy for its linked Ethereum Client and Privacy Manager.  As all incoming traffic goes through Node Hibernator, it is able to monitor the activity on the linked nodes. 
  
* Communicates with other Node Hibernators to retrieve the statuses of their linked nodes.

## Process: Hibernation of node after inactivity

![node hibernation flow](images/node-hibernation-flow.jpg)

The above sequence diagram outlines the process of hibernating a node after the configured inactivity period has been reached.  In more detail:

* **1.1:** Node Hibernator monitors the incoming requests for the linked Ethereum Client and Privacy Manager to determine whether the node is active.  The inactivity timer is reset on incoming proxy requests.

* **1.2:** If the node has been inactive for more than the configured period, Node Hibernator initiates the pre-hibernation process.

* **1.2.1 to 1.2.3:** Node Hibernator checks if the node is safe to hibernate by fetching network and consensus information from the Ethereum Client. These checks ensure the network will continue to be operational if the node hibernates. See [Consensus Checks](#Consensus-Checks) for further details.
  
  * If the validation is successful, Node Hibernator proceeds with hibernating the node.
  * If the validation is unsuccessful, Node Hibernator aborts the hibernation process, resets the inactivity timer and waits for the next inactivity trigger to attempt hibernation again.

* **1.3 to 1.3.3:** Node Hibernator checks if the node is safe to hibernate by reaching out to the other Node Hibernators in the network to see if any have also initiated hibernation of their node.  This check ensures that multiple Node Hibernators do not perform hibernation at the same time which could break the consensus checks performed earlier.
  
  * If no other Node Hibernators have initiated hibernation, Node Hibernator proceeds with hibernating the node.
  * If another Node Hibernator has initiated hibernation or did not respond, Node Hibernator aborts the hibernation process, resets the inactivity timer and waits for the next inactivity trigger to attempt hibernation again.

* **1.4:** Node Hibernator hibernates the local Ethereum Client and Privacy Manager.

### Consensus Checks

Node Hibernator ensures that hibernation would not result in a break in consensus.  The checks performed depend on the consensus mechanism of the network and whether Node Hibernator was started in `strict` mode (default) or if `disableStrictMode` was set to `true`.

| Consensus Engine | Strict Mode | Normal Mode |
| :---: | :--- | :--- |
| Raft (GoQuorum) | - **Minter** and **Peer** nodes cannot be hibernated. <br /> <br /> - **Learner** nodes can be hibernated | - **Minter** nodes cannot be hibernated <br /> <br /> - Up to ***49%*** of **Peer** nodes can be hibernated <br /> <br />- **Learner** nodes can be hibernated
| Istanbul (GoQuorum) | - **Validator** nodes cannot be hibernated <br /> <br /> - **Non-Validator** nodes can be hibernated | - Up to ***f*** **Validator** nodes can be hibernated (in a network with ***3f + 1*** Validator nodes) <br /> <br /> - **Non-Validator** nodes can be hibernated
| Clique (GoQuorum) | - **Signer** nodes cannot be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated | - Up to ***49%*** of **Signer** nodes can be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated
| Clique (Besu) | - **Signer** nodes cannot be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated | - Up to ***49%*** of **Signer** nodes can be hibernated <br /> <br /> - **Non-Signer** nodes can be hibernated

## Process: Waking of node after new activity

![request flow](images/node-hibernator-flow.jpg)

The above sequence diagram outlines the waking process for the following scenario:

1. Node *A* and Node *B* are both running Node Hibernator, GoQuorum Ethereum Client and Tessera Privacy Manager
2. All GoQuorum and Tessera nodes are hibernated due to inactivity
3. Node Hibernator *A* (acting as a proxy for GoQuorum *A*) receives a new private transaction

In more detail:

* **1.0:** Node Hibernator *A* (acting as a proxy for GoQuorum *A*) receives a new private transaction for Nodes *A* and *B*. 
  
  Node Hibernator *A* parses the transaction request:
  * As the transaction is private, Node Hibernator *A* extracts the Privacy Manager public keys from the request's `privateFor` parameter. 
  * Node Hibernator *A* then checks if the public keys match any remote Node Hibernators in its [Peers config](./config.md#Peers-config-file).  If there are no matches, it assumes that the node is not managed by a Node Hibernator.

*  **1.1:** Node Hibernator *A* check if the local GoQuorum and Tessera are up. 

* **1.2:** If the local GoQuorum or Tessera are down, Node Hibernator *A* wakes them up.

* **1.3.1 to 1.3.4:** Node Hibernator *A* asks Node Hibernator *B* for its status. Node Hibernator *B* checks the status of its linked GoQuorum and Tessera. 
  * If they are down Node Hibernator *B* initiates its wake up process. Node Hibernator *A* aborts the private transaction send. See [Understanding Client Errors](./deployment.md#Understanding-Client-Errors) for more info.
  * If they are up Node Hibernator *B* responds appropriately.  Node Hibernator *A* continues the private transaction send. 

* **1.4:** Once all nodes are up, Node Hibernator *A* forwards the request to Node *A*'s GoQuorum for processing.

* **1.4.1 to 1.4.8:** This is the standard private transaction processing flow for GoQuorum. Once the private transaction is processed, GoQuorum responds back to Node Hibernator *A* with the appropriate response.

* **1.4.9, 1.4.10:** Node Hibernator *A* receives the response for the transaction and returns it to the client.
