package main

import (
	"flag"
	"errors"
	"log"
	"net"
	"fmt"
)

type Kurafuto struct {
	Players []Player
	Hub *Server
	Config *Config
	Listener net.Listener
	Done chan bool
}

func (ku *Kurafuto) Quit() {
	ku.Done <- true
	ku.Listener.Close()
}

func (ku *Kurafuto) Run() {
	for {
		c, err := ku.Listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		Debugf("New connection: %s", c.RemoteAddr().String())

		p, err := NewPlayer(c, ku)
		if err != nil {
			c.Close()
			continue
		}
		if ku.Config.Parse {
			go p.Parse()
		} else {
			go p.Proxy()
		}
	}
}

func NewKurafuto(config *Config) (ku *Kurafuto, err error) {
	if len(config.Servers) < 1 {
		err = errors.New("kurafuto: Need at least 1 server in config.")
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Address, config.Port))
	if err != nil {
		return
	}

	ku = &Kurafuto{
		Players: []Player{},
		Hub: &config.Servers[0],
		Config: config,
		Listener: listener,
		Done: make(chan bool, 1),
	}
	return
}

var (
	Ku *Kurafuto
	debug bool
)

func Debugf(s string, v ...interface{}) {
	if !debug {
		return
	}
	log.Printf(s, v...)
}

func main() {
	var configFile = flag.String("config", "kurafuto.json", "the file your Kurafuto configuration is stored in.")
	flag.BoolVar(&debug, "debug", false, "enable verbose debugging.")
	flag.Parse()

	config, err := NewConfigFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	Ku, err := NewKurafuto(config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Kurafuto now listening on %s:%d with %d servers", config.Address, config.Port, len(config.Servers))
	Debugf("Debugging enabled!")

	go Ku.Run()
	<-Ku.Done
}
