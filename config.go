package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
)

type Config struct {
	Authenticate bool     `json:"verify-names"`
	Heartbeat    bool     `json:"heartbeat"`
	Parse        bool     `json:"parse-packets"`
	Name         string   `json:"name"`
	Motd         string   `json:"motd"`
	Address      string   `json:"address"`
	Port         int      `json:"port"`
	Servers      []Server `json:"servers"`
}

func (c *Config) Dumps() (string, error) {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type Server struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func NewConfig(r io.Reader) (*Config, error) {
	conf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var c Config
	err = json.Unmarshal(conf, &c)
	if err != nil {
		return nil, err
	}
	// Authentication requires parsing.
	if c.Authenticate {
		// Deal with it.
		c.Parse = true
	}
	return &c, nil
}

func NewConfigFile(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return NewConfig(f)
}
