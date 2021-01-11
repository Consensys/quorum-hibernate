# Deploying Node Manager

Node Manager must run on the same host as the linked Blockchain Client and Privacy Manager.

Node Manager, and the linked Blockchain Client and Privacy Manager, can be run as host processes or Docker containers.  Note that if running Node Manager as a Docker container the linked Blockchain Client and Privacy Manager must also be run as containers. 

## User Errors and Actions
User-submitted requests to Node Manager will fail in the following scenarios:

| Scenario  | Error | Action |
| --- | --- | --- |
| User sends request when Node Manager is hibernating the Blockchain Client and Privacy Manager | 500 (Internal Server Error) - `node is being shutdown, try after sometime` | Retry after some time. |  
| User sends request when Node Manager is starting the Blockchain Client and Privacy Manager | 500 (Internal Server Error) - `node is being started, try after sometime` | Retry after some time. |  
| User sends a private transaction request when at least one of the remote recipients is hibernated by Node Manager | 500 (Internal Server Error) - `Some participant nodes are down` | Retry after some time. |  
| User sends request after Node Manager has encountered an issue during hibernation/waking up of Blockchain Client or Privacy Manager | 500 (Internal Server Error) - `node is not ready to accept request` | Investigate the cause of Node Manager's failure and fix the issue. |  

*Note: Node Manager will consider a peer to be hibernated if it does not receive a response the peer's status during private transaction processing.*
