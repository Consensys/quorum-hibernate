package config

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadNodeConfig(t *testing.T) {
	fileContents := `[basicConfig]
#node manager name
name = "node1"
#blockchain client RPC URL
bcClntRpcUrl = "http://localhost:22000"
privManKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8="
consensus = "raft"
clientType = "quorum"
upcheckPollingInterval = 1
peersConfigFile = "./test/shell/nm1.toml"
#blockchain client/privacy manager inactivity timeout seconds
inactivityTime = 60
runMode = "STRICT"

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
name = "bcclnt"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"]
upcheckCfg = { upcheckUrl = "http://localhost:22000", method = "POST", body = "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}",returnType = "rpcresult"}

#privacy manager process control config
[basicConfig.privManProcess]
name = "privman"
controlType = "shell"
stopCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"]
startCommand = ["bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"]
upcheckCfg = { upcheckUrl = "http://localhost:9001/upcheck", method = "GET", body = "", returnType = "string", expected = "I'm up!"}
`

	f, err := ioutil.TempFile("", "qnmconfig")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(fileContents))
	require.NoError(t, err)

	got, err := ReadNodeConfig(f.Name())
	require.NoError(t, err)

	want := Node{
		BasicConfig: &Basic{
			Name:                 "node1",
			BcClntRpcUrl:         "http://localhost:22000",
			PrivManKey:           "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
			Consensus:            "raft",
			ClientType:           "quorum",
			UpchkPollingInterval: 1,
			PeersConfigFile:      "./test/shell/nm1.toml",
			InactivityTime:       60,
			RunMode:              "STRICT",
			Server: &RPCServer{
				RpcAddr:     "localhost:8081",
				RPCCorsList: []string{"*"},
				RPCVHosts:   []string{"*"},
			},
			BcClntProcess: &Process{
				Name:         "bcclnt",
				ControlType:  "shell",
				ContainerId:  "",
				StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"},
				StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"},
				UpcheckCfg: Upcheck{
					UpcheckUrl: "http://localhost:22000",
					Method:     "POST",
					Body:       "{\"jsonrpc\":\"2.0\", \"method\":\"eth_blockNumber\", \"params\":[], \"id\":67}",
					ReturnType: "rpcresult",
				},
			},
			PrivManProcess: &Process{
				Name:         "privman",
				ControlType:  "shell",
				ContainerId:  "",
				StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"},
				StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"},
				UpcheckCfg: Upcheck{
					UpcheckUrl: "http://localhost:9001/upcheck",
					Method:     "GET",
					Body:       "",
					ReturnType: "string",
					Expected:   "I'm up!",
				},
			},
			Proxies: []*Proxy{
				{
					Name:         "geth-rpc",
					Type:         "http",
					ProxyAddr:    "localhost:9091",
					UpstreamAddr: "http://localhost:22000",
					ProxyPaths:   []string{"/"},
					ReadTimeout:  15,
					WriteTimeout: 15,
				},
				{
					Name:         "geth-graphql",
					Type:         "http",
					ProxyAddr:    "localhost:9191",
					UpstreamAddr: "http://localhost:8547/graphql",
					ProxyPaths:   []string{"/graphql"},
					ReadTimeout:  15,
					WriteTimeout: 15,
				},
				{
					Name:         "geth-ws",
					Type:         "ws",
					ProxyAddr:    "localhost:9291",
					UpstreamAddr: "ws://localhost:23000",
					ProxyPaths:   []string{"/"},
					ReadTimeout:  15,
					WriteTimeout: 15,
				},
				{
					Name:         "tessera",
					Type:         "http",
					ProxyAddr:    "localhost:9391",
					UpstreamAddr: "http://127.0.0.1:9001",
					ProxyPaths:   []string{"/version", "/upcheck", "/resend", "/push", "/partyinfo", "/partyinfo-mirror", "/partyinfo/validate"},
					ReadTimeout:  15,
					WriteTimeout: 15,
				},
			},
		},
		NodeManagers: nil,
	}

	require.Equal(t, want, got)
}
