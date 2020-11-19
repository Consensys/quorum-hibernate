# Quorum Node Manager

Quorum Node Manger is a tool that monitors inactivity in `geth` and `tessera` and stops them when they are inactive.

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
./node-manager -config <path to toml config file>
```

sample config file

```$xslt
#node manager name
name = "node1"

#geth RPC URL
gethRpcUrl = "http://localhost:22000"
tesseraUpcheckUrl = "http://localhost:9001/upcheck"
enodeId="ac6b1096ca56b9f6d004b779ae3728bf83f8e22453404cc3cef16a3d9b96608bc67c4b30db88e0a5a6c6390213f7acbe1153ff6d23ce57380104288ae19373ef"
consensus="istanbul"

#geth inactivity timeout seconds
gethInactivityTime = 1000

#geth's http/ws services that need to be exposed as proxy services
proxies = [
    { name = "geth-rpc", type = "http", proxyAddr = "localhost:9091", upstreamAddr = "http://localhost:22000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-graphql", type = "http", proxyAddr = "localhost:9191", upstreamAddr = "http://localhost:8547/graphql", proxyPaths = ["/graphql"], readTimeout = 15, writeTimeout = 15 },
    { name = "geth-ws", type = "ws", proxyAddr = "localhost:9291", upstreamAddr = "ws://localhost:23000", proxyPaths = ["/"], readTimeout = 15, writeTimeout = 15 },
    { name = "tessera", type = "http", proxyAddr = "localhost:9391", upstreamAddr = "http://127.0.0.1:9001", proxyPaths = ["/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"], readTimeout = 15, writeTimeout = 15 },
 ]

nodeManagers = [
    { name = "node1", enodeId="ac6b1096ca56b9f6d004b779ae3728bf83f8e22453404cc3cef16a3d9b96608bc67c4b30db88e0a5a6c6390213f7acbe1153ff6d23ce57380104288ae19373ef", tesseraKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=", rpcUrl = "http://localhost:8081" },
    { name = "node2", enodeId="0ba6b9f606a43a95edc6247cdb1c1e105145817be7bcafd6b2c0ba15d58145f0dc1a194f70ba73cd6f4cdd6864edc7687f311254c7555cc32e4d45aeb1b80416", tesseraKey = "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=", rpcUrl = "http://localhost:8082" },
    { name = "node3", enodeId="579f786d4e2830bbcc02815a27e8a9bacccc9605df4dc6f20bcc1a6eb391e7225fff7cb83e5b4ecd1f3a94d8b733803f2f66b7e871961e7b029e22c155c3a778", tesseraKey = "1iTZde/ndBHvzhcl7V68x44Vx7pl8nwx9LqnM/AfJUg=", rpcUrl = "http://localhost:8083" },
    { name = "node4", enodeId="3d9ca5956b38557aba991e31cf510d4df641dce9cc26bfeb7de082f0c07abb6ede3a58410c8f249dabeecee4ad3979929ac4c7c496ad20b8cfdd061b7401b4f5", tesseraKey = "1iTZde/ndBHvzhcl7V68x44Vx7pl8nwx9LqnM/AfJUg=", rpcUrl = "http://localhost:8084" }
]

#rpc server details of node manager
[server]
# The interface + port the application should bind to
rpcAddr = "localhost:8081"
rpcCorsList = ["*"]
rpcvHosts = ["*"]

#geth's process control config
[gethProcess]
name="geth"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"]

#tessera's process control config
[tesseraProcess]
name="tessera"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"]

```