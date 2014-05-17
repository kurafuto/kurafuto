package main

import (
	"errors"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/sysr-q/kyubu/packets"
)

var ErrPacketSkipped = errors.New("kurafuto: Packet skipped")

type HookDirection int

func (h HookDirection) String() string {
	if h == FromClient {
		return "C->S"
	} else if h == FromServer {
		return "S->C"
	}
	return "?->?"
}

const (
	FromClient HookDirection = iota
	FromServer
)

// A Hook is a function which takes a packet, and information about where the
// packet came from (player and direction). A return value of `true` means the
// packet has been "handled", and the parser will skip to the next packet.
type Hook func(*Player, HookDirection, packets.Packet) bool

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
type Parser struct {
	player    *Player
	parser    packets.Parser
	hooks     map[byte][]hookInfo
	Direction HookDirection
	Disable   bool // Allows all hooks to be bypassed.
}

// Next returns the next packet parsed out of the internal parser, and fires any
// hooks related to this packet type. If any of the hooks return "handled", Next
// will return `kurafuto.ErrPacketSkipped`. Users of the parser are expected to
// re-call Next.
func (p *Parser) Next() (packets.Packet, error) {
	packet, err := p.parser.Next()
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

func NewParser(player *Player, parser packets.Parser, dir HookDirection) packets.Parser {
	return &Parser{
		player:    player,
		parser:    parser,
		hooks:     make(map[byte][]hookInfo),
		Direction: dir,
	}
}
