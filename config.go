package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type packetList []byte

func (p *packetList) String() string {
	b := []string{}
	for _, id := range *p {
		b = append(b, fmt.Sprintf("%#.2x", id))
	}
	return "[" + strings.Join(b, ", ") + "]"
}

func (p *packetList) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	b := []byte{}
	for _, s := range strings.Split(str, ",") {
		i, err := strconv.ParseInt(s, 0, 0)
		if err != nil {
			return err
		}
		b = append(b, byte(i))
	}
	*p = b
	return nil
}

func (p *packetList) MarshalJSON() ([]byte, error) {
	b := []string{}
	for _, id := range *p {
		b = append(b, fmt.Sprintf("%#.2x", id))
	}
	return []byte(`"` + strings.Join(b, ",") + `"`), nil
}

type commaString []string

func (p *commaString) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*p = strings.Split(str, ",")
	return nil
}

func (p *commaString) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strings.Join(*p, ",") + `"`), nil
}

var identRegexp = regexp.MustCompile(`^[a-zA-Z_-][a-zA-Z0-9_-]*$`)

// ident is a string that must match the regex: /^[a-zA-Z_-][a-zA-Z0-9_-]*$/
type ident string

func (p *ident) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if m := identRegexp.FindStringIndex(str); m == nil {
		return fmt.Errorf("%q is not a valid identifier", str)
	}
	*p = ident(str)
	return nil
}

func (p *ident) MarshalJSON() ([]byte, error) {
	return []byte(`"` + *p + `"`), nil
}

////////////////////

type Server struct {
	Name    ident  `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type Config struct {
	Authenticate bool     `json:"verify-names"`
	Heartbeat    bool     `json:"heartbeat"`
	EdgeCommands bool     `json:"edge-commands"`
	Name         string   `json:"name"`
	Motd         string   `json:"motd"`
	Address      string   `json:"address"`
	Port         int      `json:"port"`
	Servers      []Server `json:"servers"`

	Ignore   packetList  `json:"ignore-packets"`
	Drop     packetList  `json:"drop-packets"`
	DropExts commaString `json:"drop-extensions"`
}

func (c *Config) Dumps() (string, error) {
	b, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return "", err
	}
	return string(b), nil
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
	return &c, nil
}

func NewConfigFile(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return NewConfig(f)
}
