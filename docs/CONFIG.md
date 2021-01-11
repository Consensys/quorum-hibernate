# Node Manager

## Supported Deployment Models
Node Manager must run in the same host where block chain client and privacy manager are running. Node Manager, block chain client and privacy manger can be run as host process or docker container. The supported combination is as given below.

| Node Manager  | Blockchain Client | Privacy Manager |
| :---: | :---: | :---: |
| Host process | Host process | Host process |
| Host process | Docker | Docker |
| Docker | Docker | Docker | 
 
 
## Up & Running

### Using Binary

#### Build

```bash
go install
```

#### Run

Ensure that `node-manager` is there in `$PATH` 

```bash
node-manager --config path/to/config.json --verbosity 3
```

| Flag | Description |
| :---: | :--- |
| `--config` | Path to `.json` or `.toml` configuration file |
| `--verbosity` | Logging level (`0` = `ERROR`, `1` = `WARN`, `2` = `INFO`, `3` = `DEBUG`) |


### Using Docker

#### Build

```bash
docker build . -t node-manager
```
#### Run

Configuration files must be supplied to the Docker container. Refer to sample config files [config.toml](config.docker.local.toml) and [nodemanager.toml](nodemanger.docker.local.toml)
```bash
docker run -p <port mapping> -v /var/run/docker.sock:/var/run/docker.sock --mount type=bind,source=<path to config>,target=/config.toml node-manager:latest

```

**Example**
```bash
docker run -p 8081:8081 -p 9091:9091 -p 9391:9391 -v /var/run/docker.sock:/var/run/docker.sock --mount type=bind,source=/usr/john/node1.toml,target=/config.toml --mount type=bind,source=/usr/john/nm1.toml,target=/nm1.toml node-manager:latest -config /config.toml
```
**Note:** `-v /var/run/docker.sock:/var/run/docker.sock` is required to start/stop blockchain client/privacy manager running as docker container.


## Configuration

For starting Node Manager, two configuration files are required: [Node Manager config](#node-manager-config-file) and [peers config](#Peers-config-file). Both `json` and `toml` formats are supported.  Samples can be found in [here](../examples/README.md).

### Node Manager config file

| Field  | Type | Description |
| :---: | :---: | :--- |
| `name` | `string` | Name for the Node Manager |
| `disableStrictMode` | `bool` | Strict mode prevents blockchain client nodes involved in the consensus from being hibernated.  This protects against an essential node being shut down and preventing the chain from progressing. It is set to `false` by default. For `raft` consensus it is recommended to set it to `true` as there would be more `follower` nodes in the network. |
| `upcheckPollingInterval` | `int` | Interval (in seconds) for performing an upcheck on the blockchain client and privacy manager to determine if they have been started/stopped by a third party (i.e. not Node Manager) |
| `peersConfigFile` | `string` | Path to a [Peers config file](#Peers-config-file) |
| `inactivityTime` | `int` | Inactivity period (in seconds) to allow on either the blockchain client or privacy manager before hibernating both |
| `resyncTime` | `int` | Time (in seconds) after which a hibernating node pair should be restarted to allow the node to sync with the chain.  Regularly syncing a node with the chain during periods of inactivity will reduce the time needed to prepare the node when receiving a client request. |
| `server` | `object` | See [server](#server) |
| `proxies` | `[]object` | See [proxy](#proxy) |
| `blockchainClient` | `object` | See [blockchainClient](#blockchainClient) |
| `privacyManager` | `object` | (Optional) See [privacyManager](#privacyManager). If privacy manager is not used, this can be ignored. |

#### server

The RPC server that exposes Node Manager's API.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `rpcAddress` | `string` | Listen address for the Node Manager API |
| `rpcCorsList` | `[]string` | List of domains from which to accept cross origin requests (browser enforced) |
| `rpcvHosts` | `[]string` |  List of virtual hostnames from which to accept requests (server enforced) |
| `tlsConfig` | `object` | (Optional) See [serverTLS](#serverTLS) |

#### proxy

The proxy server for a single blockchain client or privacy manager service.  Multiple proxies can be configured.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `name` | `string` | Name of the proxy server |
| `type` | `string` | `http` or `ws` |
| `proxyAddress` | `string` | Listen address for the proxy server |
| `upstreamAddress` | `string` | Address of the blockchain client or privacy manager service |
| `proxyPaths` | `[]string` | Paths the proxy server should listen on (`/` listens on all paths) |
| `ignorePathsForActivity` | `[]string` | (Optional) Paths that should not reset the inactivity timer if called  |
| `readTimeout` | `int` | Read timeout |
| `writeTimeout` | `int` | Write timeout |
| `proxyTlsConfig` | `object` | (Optional) See [serverTLS](#serverTLS) |
| `clientTlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |

#### blockchainClient

The blockchain client to be managed by the Node Manager.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `type` | `string` | `goquorum` or `besu` |
| `consensus` | `string` | `raft`, `istanbul`, or `clique` |
| `rpcUrl` | `string` | RPC URL of blockchain client.  Used when performing consensus checks. |
| `process` | `object` | See [process](#process) |
| `tlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |

#### privacyManager

The privacy manager to be managed by the Node Manager.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `publicKey` | `string` | Privacy manager's base64-encoded public key |
| `process` | `object` | See [process](#process) |
| `tlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |

##### process

The blockchain client or privacy manager process.  Can be a standalone shell process or a Docker container. 

| Field  | Type | Description |
| :---: | :---: | :--- |
| `name` | `string` | `bcclnt` or `privman` |
| `controlType` | `string` | `shell` or `docker` |
| `containerId` | `string` | (Optional) Docker container ID.  Required if `controlType = docker` |
| `startCommand` | `[]string` | Shell command to start process.  Required if `controlType = shell` |
| `stopCommand` | `[]string` | Shell command to stop process.  Required if `controlType = shell` |
| `upcheckConfig` | `object` | See [upcheckConfig](#upcheckConfig) |

##### upcheckConfig

How Node Manager should determine whether the process is running or not.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `url` | `string` | Up check URL of the process |
| `returnType` | `string` | `string` or `rpcresult`. Provides support for REST upcheck endpoints and RPC endpoints |
| `method` | `string` | `GET` or `POST`. HTTP request method required for upcheck endpoint  |
| `body` | `string` | Body of RPC upcheck request  |
| `expected` | `string` | Expected response if `returnType = string`. |

##### serverTLS

1-way and mutual (2-way) TLS can be configured as required.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `keyFile` | `string` | Path to `.pem` encoded key file |
| `certificateFile` | `string` | Path to `.pem` encoded certificate file |
| `clientCaCertificateFile` | `string` | Path to `.pem` encoded CA certificate file to validate client |
| `cipherSuites` | `[]string` | (Optional) List of cipher suites to use in TLS.  If not set, [defaults](#cipher-suites) will be used. |

##### clientTLS

1-way and mutual (2-way) TLS can be configured as required.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `insecureSkipVerify` | `bool` | Skip verification of server certificate if `true` |
| `caCertificateFile` | `string` | Path to `.pem` encoded CA certificate file to validate server |
| `keyFile` | `string` | Path to `.pem` encoded key file |
| `certificateFile` | `string` | Path to `.pem` encoded certificate file |
| `cipherSuites` | `[]string` | (Optional) List of cipher suites to use in TLS.  If not set, [defaults](#cipher-suites) will be used. |

##### Cipher Suites
The TLS cipher suites used by default are:

* TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
* TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
* TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
* TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA

#### Peers config file

It contains list of other Node Managers in the network. This config can be updated whenever there is a change. 
It is used by Node Manager to check the status of other Node Managers when it decides to stop the nodes.
Node Manager always reads the latest information from this config before performing the checks. Any updates
 to the config file takes effect immediately.
 If there are no peers it can be empty.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `peers` | `[]object` | See [peer](#peer) for details |

##### peer

Another active Node Manager in the network.  Multiple peers can be configured.

| Field  | Type | Description |
| :--- | :---: | :--- |
| `name` | `string` | Name of the peer |
| `privacyManagerKey` | `string` | (Optional) Public key of the peer's privacy manager |
| `rpcUrl` | `string` | URL of the peer's RPC server |
| `tlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |

### Error handling for user
User requests to Node Manager will fail under the following scenarios.

| Scenario  | Error message received by user | Action required |
| --- | --- | --- |
| Node Manager receives a request from user while block chain client and privacy manager are being stopped by it due to inactivity. | 500 (Internal Server Error) - `node is being shutdown, try after sometime` | Retry after some time. |  
| Node Manager receives a request from user while block chain client and privacy manager are being started up by it due to activity. | 500 (Internal Server Error) - `node is being started, try after sometime` | Retry after some time. |  
| Node Manager receives a private transaction request from user and participant node(of the transaction) managed by Node Manager is down. | 500 (Internal Server Error) - `Some participant nodes are down` | Retry after some time. |  
| Node Manager receives a request from user when starting/stopping of block chain client or privacy manager by Node Manager failed. | 500 (Internal Server Error) - `node is not ready to accept request` | Investigate the cause of failure and fix the issue. |  

Node Manager will consider its peer is down and proceed with processing if it is not able to get a response from its peer Node Manager when it tries to check the status for stopping nodes or handling private transaction.
