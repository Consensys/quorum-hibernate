package types

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamedError(t *testing.T) {
	got := namedValidationError{name: "someName", errMsg: "someErr"}
	require.EqualError(t, got, "name = someName: someErr")
}

func TestReadNodeManagerConfig(t *testing.T) {
	fileContents := `nodeManagers = [
	{ name = "node1", privManKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=", rpcUrl = "http://localhost:8081" },
	{ name = "node2", privManKey = "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=", rpcUrl = "http://localhost:8082" }
]`

	f, err := ioutil.TempFile("", "remotesconfig")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(fileContents))
	require.NoError(t, err)

	got, err := ReadNodeManagerConfig(f.Name())
	require.NoError(t, err)

	want := []*NodeManagerConfig{
		{
			Name:       "node1",
			PrivManKey: "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
			RpcUrl:     "http://localhost:8081",
		},
		{
			Name:       "node2",
			PrivManKey: "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=",
			RpcUrl:     "http://localhost:8082",
		},
	}

	// dereference is required for require.Contains
	gotDeref := make([]NodeManagerConfig, len(got))
	for i := range got {
		gotDeref[i] = *got[i]
	}

	require.Len(t, got, 2)
	require.Contains(t, gotDeref, *want[0])
	require.Contains(t, gotDeref, *want[1])

}

func TestReadNodeConfig(t *testing.T) {
	fileContents := `[basicConfig]
#node manager name
name = "node1"
#blockchain client RPC URL
bcClntRpcUrl = "http://localhost:22000"
privManUpcheckUrl = "http://localhost:9001/upcheck"
privManKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8="
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
`

	f, err := ioutil.TempFile("", "qnmconfig")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(fileContents))
	require.NoError(t, err)

	got, err := ReadNodeConfig(f.Name())
	require.NoError(t, err)

	want := NodeConfig{
		BasicConfig: &BasicConfig{
			Name:                  "node1",
			BcClntRpcUrl:          "http://localhost:22000",
			PrivManUpcheckUrl:     "http://localhost:9001/upcheck",
			PrivManKey:            "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=",
			Consensus:             "raft",
			ClientType:            "quorum",
			NodeManagerConfigFile: "./test/shell/nm1.toml",
			InactivityTime:        60,
			Server: &RPCServerConfig{
				RpcAddr:     "localhost:8081",
				RPCCorsList: []string{"*"},
				RPCVHosts:   []string{"*"},
			},
			BcClntProcess: &ProcessConfig{
				Name:         "geth",
				ControlType:  "shell",
				ContainerId:  "",
				StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopNode.sh", "22000"},
				StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startNode.sh", "1"},
			},
			PrivManProcess: &ProcessConfig{
				Name:         "tessera",
				ControlType:  "shell",
				ContainerId:  "",
				StopCommand:  []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/stopTessera.sh", "2"},
				StartCommand: []string{"bash", "/Users/maniam/tmp/quorum-examples/examples/7nodes/startTessera.sh", "2"},
			},
			Proxies: []*ProxyConfig{
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
