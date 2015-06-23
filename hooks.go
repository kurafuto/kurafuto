package main

import (
	"fmt"
	"github.com/kurafuto/kyubu/modern"
	"github.com/kurafuto/kyubu/packets"
	"strings"
)

// LogMessage is an example hook function that simply logs all message packets
// that pass through Kurafuto. It's not that interesting, honestly.
//
//   parser := NewParser(...)
//   parser.Register(packets.Message{}, LogMessage)
func LogMessage(p *Player, dir packets.PacketDirection, packet packets.Packet) bool {
	var msg *classic.Message
	msg = packet.(*classic.Message)
	if dir == packets.ServerBound {
		Log(Colorify(fmt.Sprintf("&f<%s>&r %s", p.Name, msg.Message)))
	} else if dir == packets.ClientBound {
		Log(Colorify(fmt.Sprintf("&6[SERVER]&r %s", msg.Message)))
	} else {
		Warnf("LogMessage for %s, direction is: %d", p.Name, dir)
	}
	return false
}

// DropPacket is a simple hook which will "skip" dropped packets included in the
// server's drop list (including dropped CPE extensions).
func DropPacket(p *Player, dir packets.PacketDirection, packet packets.Packet) (drop bool) {
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
		Debugf("(%s) %s dropped packet %#.2x", p.Id, p.Name, packet.Id())
	}
	return
}

////

const (
	commandPrefix = ":kura"
	commandHelp   = "&5Type :kura list or :kura jump <server>"
)

func EdgeCommand(p *Player, dir packets.PacketDirection, packet packets.Packet) (drop bool) {
	if dir != packets.ServerBound || Ku == nil || Ku.Config == nil || !Ku.Config.EdgeCommands {
		return
	}

	var bits []string

	if msg, ok := packet.(*classic.Message); ok {
		bits = strings.Split(msg.Message, " ")
	} else {
		return
	}

	if len(bits) < 1 || bits[0] != commandPrefix {
		return
	}

	switch bits[1] {
	case "list":
		msg, _ := classic.NewMessage(127, "&5List of servers:")
		// TODO: for _, server := range Ku.Config.Servers {}
		p.Client.C <- msg
	case "info":
		// TODO: add server name, motd, + basic info.
		message := fmt.Sprintf("&5%d players are online!", len(Ku.Players))
		msg, _ := classic.NewMessage(127, message)
		p.Client.C <- msg
	case "help":
		msg, _ := classic.NewMessage(127, commandHelp)
		p.Client.C <- msg
	default:
		msg, _ := classic.NewMessage(127, commandHelp)
		p.Client.C <- msg
	}
	return true
}
