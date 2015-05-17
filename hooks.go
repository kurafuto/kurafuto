package main

import (
	"fmt"
	"github.com/sysr-q/kyubu/cpe"
	"github.com/sysr-q/kyubu/packets"
	"strings"
)

// LogMessage is an example hook function that simply logs all message packets
// that pass through Kurafuto. It's not that interesting, honestly.
//
//   parser := NewParser(...)
//   parser.Register(packets.Message{}, LogMessage)
func LogMessage(p *Player, dir HookDirection, packet packets.Packet) bool {
	var msg *packets.Message
	msg = packet.(*packets.Message)
	if dir == FromClient {
		Log(Colorify(fmt.Sprintf("&f<%s>&r %s", p.Name, msg.Message)))
	} else {
		Log(Colorify(fmt.Sprintf("&6[SERVER]&r %s", msg.Message)))
	}
	return false
}

// DropPacket is a simple hook which will "skip" dropped packets included in the
// server's drop list (including dropped CPE extensions).
func DropPacket(p *Player, dir HookDirection, packet packets.Packet) (drop bool) {
	if Ku == nil || Ku.Config == nil {
		drop = false
		return
	}
	for _, id := range Ku.Config.Drop {
		if id != packet.Id() {
			continue
		}
		drop = true
		break
	}
	if ep, ok := packet.(cpe.ExtPacket); !drop && ok {
		for _, ext := range Ku.Config.DropExts {
			if ext != ep.String() {
				continue
			}
			drop = true
			break
		}
	}
	if drop {
		Debugf("(%s) %s dropped packet %#.2x (%s)", p.Id, p.Name, packet.Id(), dir.String())
	}
	return
}

func DebugPacket(p *Player, dir HookDirection, packet packets.Packet) (drop bool) {
	if Ku == nil || Ku.Config == nil {
		return
	}
	for _, id := range Ku.Config.Ignore {
		if id != packet.Id() {
			continue
		}
		return
	}
	name := "Unknown"
	if info, ok := packets.Packets[packet.Id()]; ok {
		name = info.Name
	}
	Debugf("(%s) %s; %s packet %#.2x [%s]", p.Id, p.Name, dir.String(), packet.Id(), name)
	return
}

////

const (
	commandPrefix = ":kura"
	commandHelp   = "&5Type :kura list or :kura jump <server>"
)

func EdgeCommand(p *Player, dir HookDirection, packet packets.Packet) (drop bool) {
	if dir != FromClient || Ku == nil || Ku.Config == nil || !Ku.Config.EdgeCommands {
		return
	}

	var bits []string

	if msg, ok := packet.(*packets.Message); ok {
		bits = strings.Split(msg.Message, " ")
	} else {
		return
	}

	if len(bits) < 1 || bits[0] != commandPrefix {
		return
	}

	switch bits[1] {
	case "list":
		msg, _ := packets.NewMessage(127, "&5List of servers:")
		// TODO: for _, server := range Ku.Config.Servers {}
		p.toClient <- msg
	case "info":
		// TODO: add server name, motd, + basic info.
		message := fmt.Sprintf("&5%d players are online!", len(Ku.Players))
		msg, _ := packets.NewMessage(127, message)
		p.toClient <- msg
	case "help":
		msg, _ := packets.NewMessage(127, commandHelp)
		p.toClient <- msg
	default:
		msg, _ := packets.NewMessage(127, commandHelp)
		p.toClient <- msg
	}
	return true
}
