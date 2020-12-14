package config

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestReadPeersConfig(t *testing.T) {
	fileContents := `peers = [
	{ name = "node1", privManKey = "oNspPPgszVUFw0qmGFfWwh1uxVUXgvBxleXORHj07g8=", rpcUrl = "http://localhost:8081" },
	{ name = "node2", privManKey = "QfeDAys9MPDs2XHExtc84jKGHxZg/aj52DTh0vtA3Xc=", rpcUrl = "http://localhost:8082" }
]`

	f, err := ioutil.TempFile("", "remotesconfig")
	require.NoError(t, err)
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(fileContents))
	require.NoError(t, err)

	got, err := ReadPeersConfig(f.Name())
	require.NoError(t, err)

	want := []*Peer{
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
	gotDeref := make([]Peer, len(got))
	for i := range got {
		gotDeref[i] = *got[i]
	}

	require.Len(t, got, 2)
	require.Contains(t, gotDeref, *want[0])
	require.Contains(t, gotDeref, *want[1])

}
