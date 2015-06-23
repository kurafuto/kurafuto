package main

import (
	"errors"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/kurafuto/kyubu/packets"
	"net"
	"sync"
	"time"
)

var (
	ErrPacketSkipped  = errors.New("kurafuto: Packet skipped")
	ErrParserFinished = errors.New("kurafuto: Parser finished (timed out)")
)

// A Hook is a function which takes a packet, and information about where the
// packet came from (player and direction). A return value of `true` means the
// packet has been "handled", and the parser will skip to the next packet.
type Hook func(*Player, packets.PacketDirection, packets.Packet) bool

type hookInfo struct {
	Id string
	F  Hook
}

// AllPackets is a special sentinel type that allows registration of hooks run
// on every packet received by a hooked Parser. It may break things if there is
// a packet registered with `Id() == 0xff`.
type AllPackets struct {
}

func (p AllPackets) Id() byte {
	return 0xff
}
func (p AllPackets) Size() int {
	return 1
}
func (p AllPackets) Bytes() []byte {
	return []byte{0xff}
}

// Parser is a wrapper implementation of a Kyubu packets.Parser, which allows
// function hooks to be run when specific packets are parsed out of the stream.
// It also allows read timeouts, where if a packet isn't received in the specified
// time, the parser is "finished", and will stop consuming packets.
type Parser struct {
	player    *Player
	conn      net.Conn
	parser    packets.Parser
	hooks     map[byte][]hookInfo
	Direction packets.PacketDirection
	Disable   bool // Allows all hooks to be bypassed.

	finished bool
	mutex    sync.Mutex
	Timeout  time.Duration
}

func (p *Parser) Finish() {
	if p == nil {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.finished = true
}

// Next returns the next packet parsed out of the internal parser, and fires any
// hooks related to this packet type. If any of the hooks return "handled", Next
// will return `kurafuto.ErrPacketSkipped`. Users of the parser are expected to
// re-call Next. If the parser is "finished", or times out it will return
// `kurafuto.ErrParserFinished` forever.
func (p *Parser) Next() (packets.Packet, error) {
	if p == nil {
		return nil, ErrParserFinished
	}

	p.mutex.Lock()
	if p.finished {
		p.mutex.Unlock()
		return nil, ErrParserFinished
	}
	p.mutex.Unlock()

	// Force a deadline, this means if we don't get a response in the given
	// time, we can consider the parser "finished".
	p.conn.SetReadDeadline(time.Now().Add(p.Timeout))

	packet, err := p.parser.Next()

	if e, ok := err.(net.Error); ok && e.Timeout() {
		p.Finish()
		return nil, ErrParserFinished
	}

	// An empty Time{} indicates removing the read deadline. I think.
	// It's what Go's net/timeout_test.go does, so whatever.
	p.conn.SetReadDeadline(time.Time{})

	if packet == nil {
		return packet, err
	}

	if p.Disable {
		// Return early, we're ignoring hooks.
		return packet, err
	}

	skipPacket := func(h []hookInfo) bool {
		for _, hook := range h {
			if skip := hook.F(p.player, p.Direction, packet); skip {
				return true
			}
		}
		return false
	}

	// Run AllPacket hooks first
	if hooks, ok := p.hooks[0xff]; ok {
		if skip := skipPacket(hooks); skip {
			return packet, ErrPacketSkipped
		}
	}

	// Regular hooks for this packet
	if hooks, ok := p.hooks[packet.Id()]; ok {
		if skip := skipPacket(hooks); skip {
			return packet, ErrPacketSkipped
		}
	}

	return packet, err
}

func (p *Parser) Register(packet packets.Packet, hook Hook) (string, error) {
	if _, ok := p.hooks[packet.Id()]; !ok {
		p.hooks[packet.Id()] = []hookInfo{}
	}
	id := uniuri.NewLen(8)
	info := hookInfo{Id: id, F: hook}
	p.hooks[packet.Id()] = append(p.hooks[packet.Id()], info)
	return id, nil
}

func (p *Parser) Unregister(hookId string) (bool, error) {
	for id, hooks := range p.hooks {
		for i, hook := range hooks {
			if hook.Id != hookId {
				continue
			}
			// This just removes the hook we're looking for. Bless Golang.
			p.hooks[id] = append(p.hooks[id][:i], p.hooks[id][i+1:]...)
			return true, nil
		}
	}
	return false, fmt.Errorf("kurafuto: No hook registered for id %s", hookId)
}

// UnregisterAll forcefully unregisters all currently registered hooks by recreating
// the internal hooks list.
func (p *Parser) UnregisterAll() {
	p.hooks = make(map[byte][]hookInfo)
}

func NewParser(player *Player, conn net.Conn, dir packets.PacketDirection, t time.Duration) packets.Parser {
	return &Parser{
		player:    player,
		conn:      conn,
		parser:    packets.NewParser(conn, dir),
		hooks:     make(map[byte][]hookInfo),
		Direction: dir,
		mutex:     sync.Mutex{},
		Timeout:   t,
	}
}
