# Node Manager

Node Manger is a proxy tool that monitors inactivity in `blockchain client` and `privacy manager` and stops them when they are inactive.
When it receives request from a client, it will bring up blockchain client and privacy manager if they are down.

## Usage 

### Pre-requisites

### Up & Running

#### Using Binary

##### Build

```bash
go build [-o node-manager]
```

##### Run

- Running with a custom configuration path
```bash
./node-manager --config <path to toml config file> --verbosity <verbose level>
```

sample node config file

```$xslt
[basicConfig]
#node manager name
name = "node1"
#blockchain client RPC URL
bcClntRpcUrl = "http://localhost:22000"
privManUpcheckUrl = "http://localhost:9001/upcheck"
PrivManKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8="
consensus = "raft"
clientType = "quorum"
nodeManagerConfigFile = "./test/shell/nm1.toml"

#blockchain client/privacy manager inactivity timeout seconds
inactivityTime = 60

#blockchain client's http/ws services and privacy manager's http that need to be exposed as proxy services
proxies = [
    { name = "geth-rpc", type = "http", proxyAddr = "localhost:9091", upstreamAddr = "http://localhost:22000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-graphql", type = "http", proxyAddr = "localhost:9191", upstreamAddr = "http://localhost:8547/graphql", proxyPaths = ["/graphql"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-ws", type = "ws", proxyAddr = "localhost:9291", upstreamAddr = "ws://localhost:23000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "tessera", type = "http", proxyAddr = "localhost:9391", upstreamAddr = "http://127.0.0.1:9001", proxyPaths = ["/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"], readTimeout = 15, writeTimeout = 15 },
]

#rpc server details of node manager
[basicConfig.server]
# The interface + port the application should bind to
rpcAddr = "localhost:8081"
rpcCorsList = ["*"]
rpcvHosts = ["*"]

#blockchain client's process control config
[basicConfig.bcClntProcess]
name = "geth"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"]

#privacy manager process control config
[basicConfig.privManProcess]
name = "tessera"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"]
```

sample node manager config file

```$xslt
nodeManagers = [
    { name = "node1", privManKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=", rpcUrl = "http://localhost:8081" },
    { name = "node2", privManKey = "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=", rpcUrl = "http://localhost:8082" },
    { name = "node3", privManKey = "1iTZde/ndBHvzhcl7V68x44Vx7pl8nwx9LqnM/AfJUg=", rpcUrl = "http://localhost:8083" },
    { name = "node4", privManKey = "1iTZde/ndBHvzhcl7V68x44Vx7pl8nwx9LqnM/AfJUg=", rpcUrl = "http://localhost:8084" }
]
```
