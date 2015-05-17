package main

import (
	"errors"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/sysr-q/kyubu/packets"
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
	Running  bool

	rMut sync.Mutex
}

func (ku *Kurafuto) Quit() {
	ku.rMut.Lock()
	if !ku.Running {
		ku.rMut.Unlock()
		return
	}

	ku.Running = false
	ku.rMut.Unlock()

	// So we don't take on any new players.
	ku.Listener.Close()
	for len(ku.Players) <= 0 {
		for _, p := range ku.Players {
			disc, _ := packets.NewDisconnectPlayer("Server shutting down.")
			p.toClient <- disc
			p.Quit()
		}
	}
	ku.Done <- true
}

func (ku *Kurafuto) Run() {
	ku.rMut.Lock()
	ku.Running = true
	ku.rMut.Unlock()

	for {
		ku.rMut.Lock()
		if !ku.Running {
			ku.rMut.Unlock()
			break
		}
		ku.rMut.Unlock()

		c, err := ku.Listener.Accept()
		if err != nil && !ku.Running {
			break
		} else if err != nil {
			Fatal(err)
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
		f := "%s (%s) disconnected"
		if p.Name == "" {
			f = "%s(%s) disconnected"
		}
		Infof(f, p.Name, p.Client.RemoteAddr().String())
		Debugf("(%s) %s disconnected from slot %d", p.Id, p.Client.RemoteAddr().String(), i)
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

		rMut: sync.Mutex{},
	}
	return
}
