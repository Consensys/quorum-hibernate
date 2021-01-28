# Configuration

For starting Node Manager, two configuration files are required: [Node Manager config](#node-manager-config-file) and [peers config](#Peers-config-file). Both `json` and `toml` formats are supported.  Sample configurations can be found in [samples](samples).

## Node Manager config file

| Field  | Type | Description |
| :---: | :---: | :--- |
| `name` | `string` | Name for the Node Manager |
| `disableStrictMode` | `bool` | Strict mode prevents Ethereum Client nodes involved in the consensus from being hibernated.  This protects against an essential node being shut down and preventing the chain from progressing. It is set to `false` by default. For `raft` consensus it is recommended to set it to `true` as there would be more `follower` nodes in the network. |
| `upcheckPollingInterval` | `int` | Interval (in seconds) for performing an upcheck on the Ethereum Client and Privacy Manager to determine if they have been started/stopped by a third party (i.e. not Node Manager) |
| `peersConfigFile` | `string` | Path to a [Peers config file](#Peers-config-file) |
| `inactivityTime` | `int` | Inactivity period (in seconds) to allow on either the Ethereum Client or Privacy Manager before hibernating both |
| `resyncTime` | `int` | Time (in seconds) after which a hibernating node pair should be restarted to allow the node to sync with the chain.  Regularly syncing a node with the chain during periods of inactivity will reduce the time needed to prepare the node when receiving a client request. |
| `server` | `object` | See [server](#server) |
| `proxies` | `[]object` | See [proxy](#proxy) |
| `blockchainClient` | `object` | See [blockchainClient](#blockchainClient) |
| `privacyManager` | `object` | (Optional) See [privacyManager](#privacyManager). If Privacy Manager is not used, this can be ignored. |

### server

The RPC server that exposes Node Manager's API.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `rpcAddress` | `string` | Listen address for the Node Manager API |
| `rpcCorsList` | `[]string` | List of domains from which to accept cross origin requests (browser enforced) |
| `rpcvHosts` | `[]string` |  List of virtual hostnames from which to accept requests (server enforced) |
| `tlsConfig` | `object` | (Optional) See [serverTLS](#serverTLS) |

### proxy

The proxy server for a single Ethereum Client or Privacy Manager service.  Multiple proxies can be configured.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `name` | `string` | Name of the proxy server |
| `type` | `string` | `http` or `ws` |
| `proxyAddress` | `string` | Listen address for the proxy server |
| `upstreamAddress` | `string` | Address of the Ethereum Client or Privacy Manager service |
| `proxyPaths` | `[]string` | Paths the proxy server should listen on (`/` listens on all paths) |
| `ignorePathsForActivity` | `[]string` | (Optional) Paths that should not reset the inactivity timer if called  |
| `readTimeout` | `int` | Read timeout |
| `writeTimeout` | `int` | Write timeout |
| `proxyTlsConfig` | `object` | (Optional) See [serverTLS](#serverTLS) |
| `clientTlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |

### blockchainClient

The Ethereum Client to be managed by the Node Manager.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `type` | `string` | `goquorum` or `besu` |
| `consensus` | `string` | `raft`, `istanbul`, or `clique` |
| `rpcUrl` | `string` | RPC URL of Ethereum Client.  Used when performing consensus checks. |
| `process` | `object` | See [process](#process) |
| `tlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |

### privacyManager

The Privacy Manager to be managed by the Node Manager.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `publicKey` | `string` | Privacy manager's base64-encoded public key |
| `process` | `object` | See [process](#process) |
| `tlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |

### process

The Ethereum Client or Privacy Manager process.  Can be a standalone shell process or a Docker container. 

| Field  | Type | Description |
| :---: | :---: | :--- |
| `name` | `string` | `bcclnt` or `privman` |
| `controlType` | `string` | `shell` or `docker` |
| `containerId` | `string` | (Optional) Docker container ID.  Required if `controlType = docker` |
| `startCommand` | `[]string` | Shell command to start process.  Required if `controlType = shell` |
| `stopCommand` | `[]string` | Shell command to stop process.  Required if `controlType = shell` |
| `upcheckConfig` | `object` | See [upcheckConfig](#upcheckConfig) |

### upcheckConfig

How Node Manager should determine whether the process is running or not.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `url` | `string` | Up check URL of the process |
| `returnType` | `string` | `string` or `rpcresult`. Provides support for REST upcheck endpoints and RPC endpoints |
| `method` | `string` | `GET` or `POST`. HTTP request method required for upcheck endpoint  |
| `body` | `string` | Body of RPC upcheck request  |
| `expected` | `string` | Expected response if `returnType = string`. |

### serverTLS

1-way and mutual (2-way) TLS can be configured as required.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `keyFile` | `string` | Path to `.pem` encoded key file |
| `certificateFile` | `string` | Path to `.pem` encoded certificate file |
| `clientCaCertificateFile` | `string` | Path to `.pem` encoded CA certificate file to validate client |
| `cipherSuites` | `[]string` | (Optional) List of cipher suites to use in TLS.  If not set, [defaults](#cipher-suites) will be used. |

### clientTLS

1-way and mutual (2-way) TLS can be configured as required.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `insecureSkipVerify` | `bool` | Skip verification of server certificate if `true` |
| `caCertificateFile` | `string` | Path to `.pem` encoded CA certificate file to validate server |
| `keyFile` | `string` | Path to `.pem` encoded key file |
| `certificateFile` | `string` | Path to `.pem` encoded certificate file |
| `cipherSuites` | `[]string` | (Optional) List of cipher suites to use in TLS.  If not set, [defaults](#cipher-suites) will be used. |

#### Cipher Suites
The TLS cipher suites used by default are:

* TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
* TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
* TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
* TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA

## Peers config file

It contains list of other Node Managers in the network. This config can be updated whenever there is a change. 
It is used by Node Manager to check the status of other Node Managers when it decides to stop the nodes.
Node Manager always reads the latest information from this config before performing the checks. Any updates
 to the config file takes effect immediately.
 If there are no peers it can be empty.

| Field  | Type | Description |
| :---: | :---: | :--- |
| `peers` | `[]object` | See [peer](#peer) for details |

### peer

Another active Node Manager in the network.  Multiple peers can be configured.

| Field  | Type | Description |
| :--- | :---: | :--- |
| `name` | `string` | Name of the peer |
| `privacyManagerKey` | `string` | (Optional) Public key of the peer's Privacy Manager |
| `rpcUrl` | `string` | URL of the peer's RPC server |
| `tlsConfig` | `object` | (Optional) See [clientTLS](#clientTLS) |
