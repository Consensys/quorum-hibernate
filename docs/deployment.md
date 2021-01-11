# Deploying and Using Node Manager

## Deployment guidelines

* Node Manager must run on the same host as the linked Blockchain Client and Privacy Manager.

* Node Manager can be run as a shell process or a Docker container.  See (../README.md) for run commands.  This determines the available [process configuration](../config.md#process) options:
    * Shell process Node Manager: Can manage `shell` & `docker` Blockchain Client and Privacy Manager processes.
    * Docker container Node Manager: Can manage *only* `docker` Blockchain Client and Privacy Manager processes.

* Node Manager does not require a Privacy Manager.  All Privacy Manager related [config](../config.md) fields are optional.

## Adding Node Manager to an existing deployment

1. Construct the Node Manager config as required by the existing deployment 
1. Start Node Manager
1. If using Tessera: Update Tessera's server configs so that `serverAddress` is the corresponding Node Manager proxy address, and `bindingAddress` is the "internal" address that Node Manager will forward requests to. See [Tessera's Server Addresses docs](https://docs.tessera.consensys.net/en/latest/HowTo/Configure/TesseraAPI/#server-addresses) for more info. 
1. Update/inform clients to use the proxy addresses for all requests.  
   
**If clients continue to use the direct Blockchain Client and Privacy Manager API addresses instead of Node Manager's proxy addresses, Node Manager will be unable to accurately determine activity. This will likely lead to inconsistent behaviour.**



## Understanding Client Errors
The following table describes scenarios where user-submitted requests are expected to fail.  The Action describes the necessary steps to continue:

| Scenario  | Error | Action |
| --- | --- | --- |
| User sends request when Node Manager is hibernating the Blockchain Client and Privacy Manager | 500 (Internal Server Error) - `node is being shutdown, try after sometime` | Retry after some time. |  
| User sends request when Node Manager is starting the Blockchain Client and Privacy Manager | 500 (Internal Server Error) - `node is being started, try after sometime` | Retry after some time. |  
| User sends a private transaction request when at least one of the remote recipients is hibernated by Node Manager | 500 (Internal Server Error) - `Some participant nodes are down` | Retry after some time. |  
| User sends request after Node Manager has encountered an issue during hibernation/waking up of Blockchain Client or Privacy Manager | 500 (Internal Server Error) - `node is not ready to accept request` | Investigate the cause of Node Manager's failure and fix the issue. |  

*Note: Node Manager will consider a peer to be hibernated if it does not receive a response the peer's status during private transaction processing.*
