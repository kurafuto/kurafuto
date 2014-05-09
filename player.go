package main

import (
	"crypto/md5"
	"crypto/subtle"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/sysr-q/kyubu/packets"
	_ "github.com/sysr-q/kyubu/cpe"
	"io"
	"log"
	"net"
	"time"
)

type State int

const (
	Dead State = iota
	Identification
	Idle
)

// compareHash compares a player's given "MpPass" against the computed hash
// using the server's salt and the player's username. It uses crypto/subtle
// to avoid any super-easy timing attacks.
func compareHash(salt, name, mpPass string) bool {
	h := md5.New()
	h.Write([]byte(salt))
	h.Write([]byte(name))
	sum := fmt.Sprintf("%x", h.Sum(nil))
	Debugf("salt:%s name:%s mpPass:%s sum:%s", salt, name, mpPass, sum)
	return subtle.ConstantTimeCompare([]byte(sum), []byte(mpPass)) == 1
}

type Player struct {
	Id   string
	Name string // From 0x00 Identification packet

	Client   net.Conn       // Client <-> Balancer
	client   packets.Parser // C <-> B
	toClient chan packets.Packet

	Server   net.Conn       // Balancer <-> Server
	server   packets.Parser // B <-> S
	toServer chan packets.Packet

	State          State
	quit, quitting bool
	hub            string
	ku             *Kurafuto
}

func (p *Player) Quit() {
	if p.quit || p.quitting {
		return
	}
	p.quitting = true
	p.State = Dead
	Debugf("[%s] Remove(p) == %v", p.Id, p.ku.Remove(p))
	go func() {
		// Wait a second to write any packets still in the queue.
		time.Sleep(1 * time.Second)
		p.quit = true
		p.Client.Close()
		p.Server.Close()
		close(p.toClient)
		close(p.toServer)
	}()
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
	defer func() {
		err := recover()
		if !p.quit && err != nil {
			panic(err)
		}
	}()

	for !p.quit {
		packet, err := parser.Next()
		if packet == nil || err != nil {
			Debugf("[%s] readParse(): packet:%+v, err:%#v", p.Id, packet, err)
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
			Debugf("[%s] writeParse(): nil packet", p.Id)
			p.Quit()
			return
		}
		Debugf("[%s] -> Packet %#.2x [%s]", p.Id, packet.Id(), packets.Packets[packet.Id()].Name)

		n, err := conn.Write(packet.Bytes())
		if err != nil {
			Debugf("[%s] writeParse(): conn.Write err: %#v", p.Id, err)
			p.Quit()
			return
		}
		if n != packet.Size() {
			Debugf("[%s] Packet %#.2x is %d bytes, but %d was written", p.Id, packet.Id(), packet.Size(), n)
		}
	}
}

func (p *Player) Parse() {
	// TODO: Dial in a goroutine so that we can parse the client's Identification
	// in the mean time.
	if !p.Dial() {
		return
	}
	Debugf("[%s] Dialed %s!", p.Id, p.Server.RemoteAddr().String())
	p.client = packets.NewParser(p.Client)
	p.server = packets.NewParser(p.Server)

	// We handle authentication here if it's enabled, otherwise we just
	// pull out the username from the initial Identification packet.
	packet, err := p.client.Next()
	Debugf("[%s] %#v", p.Id, packet)
	if packet == nil || err != nil || packet.Id() != 0x00 {
		// 0x00 = Identification
		Debugf("[%s] !ident: packet:%#v err:%#v", packet, err)
		p.Quit()
		return
	}

	// We'll pass it on eventually.
	p.toServer <- packet
	go p.readParse(p.client, p.toServer)  // C -> B
	go p.writeParse(p.toClient, p.Client) // C <- B

	// Store their username!
	var ident *packets.Identification
	ident = packet.(*packets.Identification)
	p.Name = ident.Name

	if p.ku.Config.Authenticate && !compareHash(p.ku.salt, p.Name, ident.KeyMotd) {
		log.Printf("[%s] Connected, but didn't verify for %s", p.Client.RemoteAddr().String(), p.Name)
		disc, err := packets.NewDisconnectPlayer("Name wasn't verified!")
		if err != nil {
			p.Quit()
			return
		}
		p.toClient <- disc
		p.Quit()
		return
	}

	// Now we can pass it along to the server.
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
		toClient: make(chan packets.Packet, 64),
		toServer: make(chan packets.Packet, 64),
	}
	return
}
