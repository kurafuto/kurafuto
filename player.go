package main

import (
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/sysr-q/kyubu/packets"
	"io"
	"net"
)

type State int

const (
	Dead State = iota
	Identification
	Idle
)

type Player struct {
	Id string

	Client   net.Conn       // Client <-> Balancer
	client   packets.Parser // C <-> B
	toClient chan packets.Packet

	Server   net.Conn       // Balancer <-> Server
	server   packets.Parser // B <-> S
	toServer chan packets.Packet

	State State
	quit  bool
	hub   string
	ku    *Kurafuto
}

func (p *Player) Quit() {
	if p.quit {
		return
	}
	p.quit = true
	p.State = Dead
	p.Client.Close()
	p.Server.Close()
	close(p.toClient)
	close(p.toServer)
	p.ku.Remove(p)
}

// Dial (attempts to) make an outbound connection to the stored hub address.
func (p *Player) Dial() bool {
	server, err := net.Dial("tcp", p.hub)
	if err != nil {
		p.Quit()
		return false
	}
	p.Server = server
	p.State = Identification
	return true
}

func (p *Player) Redirect(address string, port int) error {
	// TODO: Close p.Server, reopen, reboot write() loop.
	// This will mean we'll have to buffer packets to retransmit, won't we?
	return nil
}

func (p *Player) readParse(parser packets.Parser, to chan packets.Packet) {
	for !p.quit {
		packet, err := parser.Next()
		if packet == nil || err != nil {
			p.Quit()
			return
		}

		to <- packet
	}
}

func (p *Player) writeParse(pack <-chan packets.Packet, conn net.Conn) {
	for !p.quit {
		packet := <-pack
		if packet == nil {
			p.Quit()
			return
		}
		Debugf("[%s] -> Packet %#.2x [%s]", p.Id, packet.Id(), packets.Packets[packet.Id()].Name)

		n, err := conn.Write(packet.Bytes())
		if err != nil {
			p.Quit()
			return
		}
		if n != packet.Size() {
			Debugf("[%s] Packet %#.2x is %d bytes, but %d was written", p.Id, packet.Id(), packet.Size(), n)
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

	go p.readParse(p.client, p.toServer)  // C -> B
	go p.writeParse(p.toClient, p.Client) // C <- B
	go p.readParse(p.server, p.toClient)  // B <- S
	go p.writeParse(p.toServer, p.Server) // B -> S
}

func (p *Player) proxyCopy(in io.Reader, out io.Writer) {
	for !p.quit {
		n, err := io.Copy(out, in)
		if err != nil {
			p.Quit()
			return
		}
		if n == 0 {
			p.Quit()
			return
		}

		Debugf("[%s] Copied %d", p.Id, n)
	}
}

func (p *Player) Proxy() {
	if !p.Dial() {
		return
	}
	// We're ignorant to states aside from Idle and Dead.
	p.State = Idle
	go p.proxyCopy(p.Client, p.Server) // C -> B -> S
	go p.proxyCopy(p.Server, p.Client) // S -> B -> C
}

func NewPlayer(c net.Conn, ku *Kurafuto) (p *Player, err error) {
	p = &Player{
		Id:       uniuri.NewLen(8),
		Client:   c,
		ku:       ku,
		hub:      fmt.Sprintf("%s:%d", ku.Hub.Address, ku.Hub.Port),
		toClient: make(chan packets.Packet),
		toServer: make(chan packets.Packet),
	}
	return
}
