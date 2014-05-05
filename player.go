package main

import (
	"net"
	"github.com/sysr-q/kyubu/packets"
	"fmt"
)

type Player struct {
	Client  net.Conn       // Client <-> Balancer
	client  packets.Parser // C <-> B
	toClient chan packets.Packet

	Server net.Conn       // Balancer <-> Server
	server packets.Parser // B <-> S
	toServer chan packets.Packet

	quit bool
	hub string
}

func (p *Player) Quit() {
	if p.quit {
		return
	}
	p.quit = true
	p.Client.Close()
	p.Server.Close()
	close(p.toClient)
	close(p.toServer)
}

// Dial (attempts to) make an outbound connection to the stored hub address.
func (p *Player) Dial() bool {
	server, err := net.Dial("tcp", p.hub)
	if err != nil {
		p.Quit()
		return false
	}
	p.Server = server
	return true
}

func (p *Player) Redirect(address string, port int) error {
	// TODO: Close p.Server, reopen, reboot write() loop.
	// This will mean we'll have to buffer packets to retransmit, won't we?
	return nil
}

func (p *Player) read(parser packets.Parser, to chan packets.Packet) {
	for !p.quit {
		packet, err := parser.Next()
		if packet == nil || err != nil {
			p.Quit()
			return
		}

		to <- packet
	}
}

func (p *Player) write(pack <-chan packets.Packet, conn net.Conn) {
	for !p.quit {
		packet := <-pack
		if packet == nil {
			p.Quit()
			return
		}
		Debugf("Sending Packet %#.2x [%s] to %s", packet.Id(), packets.Packets[packet.Id()].Name, conn.RemoteAddr().String())

		n, err := conn.Write(packet.Bytes())
		if err != nil {
			p.Quit()
			return
		}
		if n != packet.Size() {
			Debugf("packet %#.2x is %d bytes, but %d was written", packet.Id(), packet.Size(), n)
		}
	}
}

func (p *Player) Parse() {
	if !p.Dial() {
		return
	}
	// TODO: Handle authentication.
	p.client = packets.NewParser(p.Client)
	p.server = packets.NewParser(p.Server)

	go p.read(p.client, p.toServer)  // C -> B
	go p.write(p.toClient, p.Client) // C <- B
	go p.read(p.server, p.toClient)  // B <- S
	go p.write(p.toServer, p.Server) // B -> S
}

func (p *Player) Proxy() {
	if !p.Dial() {
		return
	}
}

func NewPlayer(c net.Conn, ku *Kurafuto) (p *Player, err error) {
	p = &Player{
		Client: c,
		hub: fmt.Sprintf("%s:%d", ku.Hub.Address, ku.Hub.Port),
		toClient: make(chan packets.Packet),
		toServer: make(chan packets.Packet),
	}
	return
}

