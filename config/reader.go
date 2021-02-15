package config

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/naoina/toml"
)

type NodeHibernatorReader interface {
	Read() (Basic, error)
}

type PeersReader interface {
	Read() (PeerArr, error)
}

func NewNodeHibernatorReader(f string) (NodeHibernatorReader, error) {
	if strings.HasSuffix(f, ".toml") {
		return tomlNodeHibernatorReader{file: f}, nil
	} else if strings.HasSuffix(f, ".json") {
		return jsonNodeHibernatorReader{file: f}, nil
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

type tomlNodeHibernatorReader struct {
	file string
}

func (r tomlNodeHibernatorReader) Read() (Basic, error) {
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

type jsonNodeHibernatorReader struct {
	file string
}

func (r jsonNodeHibernatorReader) Read() (Basic, error) {
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
	var input NodeHibernatorList
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
	var input NodeHibernatorList
	if err = json.NewDecoder(f).Decode(&input); err != nil {
		return nil, err
	}

	return input.Peers, nil
}
