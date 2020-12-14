package config

import (
	"encoding/json"
	"errors"
	"github.com/naoina/toml"
	"os"
	"strings"
)

type NodeManagerReader interface {
	Read() (Node, error)
}

func NewNodeManagerReader(f string) (NodeManagerReader, error) {
	if strings.HasSuffix(f, ".toml") {
		return tomlNodeManagerReader{file: f}, nil
	} else if strings.HasSuffix(f, ".json") {
		return jsonNodeManagerReader{file: f}, nil
	}
	return nil, errors.New("unsupported config file format")
}

type tomlNodeManagerReader struct {
	file string
}

func (r tomlNodeManagerReader) Read() (Node, error) {
	f, err := os.Open(r.file)
	if err != nil {
		return Node{}, err
	}
	defer f.Close()
	var input Node
	if err = toml.NewDecoder(f).Decode(&input); err != nil {
		return Node{}, err
	}

	return input, nil
}

type jsonNodeManagerReader struct {
	file string
}

func (r jsonNodeManagerReader) Read() (Node, error) {
	f, err := os.Open(r.file)
	if err != nil {
		return Node{}, err
	}
	defer f.Close()
	var input Node
	if err = json.NewDecoder(f).Decode(&input); err != nil {
		return Node{}, err
	}

	return input, nil
}

type PeersReader interface {
	Read() (PeerArr, error)
}

func NewPeersReader(f string) (PeersReader, error) {
	if strings.HasSuffix(f, ".toml") {
		return tomlPeersReader{file: f}, nil
	} else if strings.HasSuffix(f, ".json") {
		return jsonPeersReader{file: f}, nil
	}
	return nil, errors.New("unsupported config file format")
}

type tomlPeersReader struct {
	file string
}

func (r tomlPeersReader) Read() (PeerArr, error) {
	f, err := os.Open(r.file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var input NodeManagerList
	if err = toml.NewDecoder(f).Decode(&input); err != nil {
		return nil, err
	}

	return input.Peers, nil
}

type jsonPeersReader struct {
	file string
}

func (r jsonPeersReader) Read() (PeerArr, error) {
	f, err := os.Open(r.file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var input NodeManagerList
	if err = json.NewDecoder(f).Decode(&input); err != nil {
		return nil, err
	}

	return input.Peers, nil
}
