package main

import (
	"errors"
	"fmt"
	"github.com/dchest/uniuri"
	"log"
	"net"
	"sync"
	"github.com/sysr-q/kyubu/packets"
	"time"
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
	Running  bool
}

func (ku *Kurafuto) Quit() {
	ku.Running = false
	for _, p := range ku.Players {
		disc, _ := packets.NewDisconnectPlayer("Server shutting down.")
		p.toClient <- disc
		p.Quit()
	}
	ku.Listener.Close()
	// TODO: `while len(ku.Players) > 0 {}`?
	go func() {
		time.Sleep(2 * time.Second)
		ku.Done <- true
	}()
}

func (ku *Kurafuto) Run() {
	ku.Running = true
	for ku.Running {
		c, err := ku.Listener.Accept()
		if err != nil && !ku.Running {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		p, err := NewPlayer(c, ku)
		if err != nil {
			c.Close()
			continue
		}
		ku.Players = append(ku.Players, p)

		Infof("New connection from %s (%d clients)", c.RemoteAddr().String(), len(ku.Players))
		Debugf("(%s) New connection from %s", p.Id, c.RemoteAddr().String())

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
		Infof("%s (%s) disconnected", p.Name, p.Client.RemoteAddr().String())
		Debugf("(%s) Disconnected %s from slot %d", p.Id, p.Client.RemoteAddr().String(), i)
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
