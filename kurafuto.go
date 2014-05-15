package main

import (
	"errors"
	"fmt"
	"github.com/dchest/uniuri"
	"log"
	"net"
	"sync"
)

type Kurafuto struct {
	Players []*Player
	mutex   sync.Mutex

	salt string
	Name string
	Motd string

	Hub    *Server
	Config *Config

	Listener net.Listener
	Done     chan bool
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

		p, err := NewPlayer(c, ku)
		if err != nil {
			c.Close()
			continue
		}
		ku.Players = append(ku.Players, p)

		log.Printf("New connection from %s (%d clients)", c.RemoteAddr().String(), len(ku.Players))
		Debugf(1, "[%s] New connection from %s", p.Id, c.RemoteAddr().String())

		go p.Parse()
	}
}

func (ku *Kurafuto) Remove(p *Player) bool {
	ku.mutex.Lock()
	defer ku.mutex.Unlock()
	for i, player := range ku.Players {
		if player != p {
			continue
		}
		p.Quit() // just in case
		// Remove and zero player to allow GC to collect it.
		copy(ku.Players[i:], ku.Players[i+1:])
		ku.Players[len(ku.Players)-1] = nil
		ku.Players = ku.Players[:len(ku.Players)-1]
		log.Printf("%s (%s) disconnected", p.Name, p.Client.RemoteAddr().String())
		Debugf(1, "[%s] Disconnected %s from slot %d", p.Id, p.Client.RemoteAddr().String(), i)
		return true
	}
	return false
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
		Players:  []*Player{},
		mutex:    sync.Mutex{},
		salt:     uniuri.New(),
		Hub:      &config.Servers[0],
		Config:   config,
		Listener: listener,
		Done:     make(chan bool, 1),
	}
	return
}