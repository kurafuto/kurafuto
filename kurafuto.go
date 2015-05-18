package main

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/dchest/uniuri"
	"github.com/sysr-q/kyubu/classic"
)

// Signal is a type that embeds sync/atomic.Value; it's used to do thread-safe
// tests of when the server has stopped. Basically a big old thread-safe bool.
type Signal struct {
	v  atomic.Value
	mu sync.Mutex
}

// Start "restarts" the signal, by storing true.
func (s *Signal) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v.Store(true)
}

// Finish "stops" the signal, by storing false.
func (s *Signal) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.v.Store(false)
}

// Value returns the stored value of the Signal.
func (s *Signal) Value() bool {
	return s.v.Load().(bool)
}

func NewSignal() *Signal {
	s := &Signal{
		v:  atomic.Value{},
		mu: sync.Mutex{},
	}
	s.Start()
	return s
}

//////////

type Kurafuto struct {
	Players []*Player
	mutex   sync.Mutex

	salt string
	Name string
	Motd string

	Hub    *Server
	Config *Config

	Listener net.Listener

	Alive *Signal
	done  chan bool   // The channel we really send to.
	Done  <-chan bool // The channel we send to when it's all over.
}

func (ku *Kurafuto) Quit() {
	if !ku.Alive.Value() {
		return
	}

	// So we don't take on any new players.
	ku.Listener.Close()
	for len(ku.Players) > 0 {
		for _, p := range ku.Players {
			disc, _ := classic.NewDisconnectPlayer("Server shutting down.")
			p.Client.C <- disc
			p.Quit()
		}
	}

	ku.done <- true
	ku.Alive.Finish()
}

func (ku *Kurafuto) Run() {
	for ku.Alive.Value() {
		c, err := ku.Listener.Accept()
		if err != nil && !ku.Alive.Value() {
			break
		} else if err != nil {
			Fatal(err)
		}

		p, err := NewPlayer(c, ku)
		if err != nil {
			c.Close()
			continue
		}

		// Thread-safe, ayy.
		ku.mutex.Lock()
		ku.Players = append(ku.Players, p)
		ku.mutex.Unlock()

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
		Log(f, p.Name, p.Remote())
		Debugf("(%s) %s disconnected from slot %d", p.Id, p.Remote(), i)
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

	done := make(chan bool, 1)

	ku = &Kurafuto{
		Players: []*Player{},
		mutex:   sync.Mutex{},

		salt: uniuri.New(),

		Hub:      &config.Servers[0],
		Config:   config,
		Listener: listener,

		Alive: NewSignal(),

		done: done,
		Done: done,
	}
	return
}
