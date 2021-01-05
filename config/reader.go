package config

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/naoina/toml"
)

type NodeManagerReader interface {
	Read() (Basic, error)
}

type PeersReader interface {
	Read() (PeerArr, error)
}

func NewNodeManagerReader(f string) (NodeManagerReader, error) {
	if strings.HasSuffix(f, ".toml") {
		return tomlNodeManagerReader{file: f}, nil
	} else if strings.HasSuffix(f, ".json") {
		return jsonNodeManagerReader{file: f}, nil
	}
	return nil, errors.New("unsupported config file format")
}

func NewPeersReader(f string) (PeersReader, error) {
	if strings.HasSuffix(f, ".toml") {
		return tomlPeersReader{file: f}, nil
	} else if strings.HasSuffix(f, ".json") {
		return jsonPeersReader{file: f}, nil
	}
	return nil, errors.New("unsupported config file format")
}

type tomlNodeManagerReader struct {
	file string
}

func (r tomlNodeManagerReader) Read() (Basic, error) {
	f, err := os.Open(r.file)
	if err != nil {
		return Basic{}, err
	}
	defer f.Close()
	var input Basic
	if err = toml.NewDecoder(f).Decode(&input); err != nil {
		return Basic{}, err
	}

	return input, nil
}

type jsonNodeManagerReader struct {
	file string
}

func (r jsonNodeManagerReader) Read() (Basic, error) {
	f, err := os.Open(r.file)
	if err != nil {
		return Basic{}, err
	}
	defer f.Close()
	var input Basic
	if err = json.NewDecoder(f).Decode(&input); err != nil {
		return Basic{}, err
	}

	return input, nil
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
