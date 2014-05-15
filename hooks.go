package main

import (
	"fmt"
	"github.com/sysr-q/kyubu/packets"
	"log"
)

// LogMessage is an example hook function that simply logs all message packets
// that pass through Kurafuto. It's not that interesting, honestly.
func LogMessage(p *Player, dir HookDirection, packet packets.Packet) bool {
	var msg *packets.Message
	msg = packet.(*packets.Message)
	if dir == FromClient {
		log.Println(Colorify(fmt.Sprintf("&f<%s>&r %s", p.Name, msg.Message)))
	} else {
		log.Println(Colorify(fmt.Sprintf("&6[SERVER]&r %s", msg.Message)))
	}
	return false
}
