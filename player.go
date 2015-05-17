package main

import (
	"crypto/md5"
	"crypto/subtle"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dchest/uniuri"
	"github.com/sysr-q/kyubu/classic"
	"github.com/sysr-q/kyubu/packets"
)

type PlayerState int

const (
	Connecting PlayerState = iota
	Identification
	Idle // this means we're just proxying packets for this user now.
	Disconnected
)

// compareHash compares a player's given "MpPass" against the computed hash
// using the server's salt and the player's username. It uses crypto/subtle
// to avoid any super-easy timing attacks.
func compareHash(salt, name, mpPass string) bool {
	h := md5.New()
	h.Write([]byte(salt))
	h.Write([]byte(name))
	sum := fmt.Sprintf("%x", h.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(sum), []byte(mpPass)) == 1
}

type Player struct {
	Id   string
	Name string // From 0x00 Identification packet
	CPE  bool   // Does this player claim CPE support?

	Client   net.Conn // Client <-> Balancer
	client   *Parser  // C <-> B
	toClient chan packets.Packet

	Server   net.Conn // Balancer <-> Server
	server   *Parser  // B <-> S
	toServer chan packets.Packet

	State          PlayerState
	quit, quitting bool
	hub            string
	ku             *Kurafuto

	qMutex sync.Mutex
}

// Remote returns a player's remote address (connecting IP) as a string.
func (p *Player) Remote() string {
	return p.Client.RemoteAddr().String()
}

func (p *Player) Quit() {
	p.qMutex.Lock()
	if p.quit || p.quitting {
		p.qMutex.Unlock()
		return
	}

	p.quitting = true
	p.qMutex.Unlock()

	p.State = Disconnected
	rem := p.ku.Remove(p) // Ensure we're removed from the server's player list
	Debugf("(%s) Remove(p) == %v", p.Id, rem)

	go func() {
		// Wait a bit to write any packets still in the queue.
		time.Sleep(300 * time.Millisecond)

		p.qMutex.Lock()
		p.quit = true
		p.qMutex.Unlock()

		p.client.Finish()
		p.server.Finish()

		// p.client and p.server might be nil if we hit an error whilst
		// dialing the remote.
		if p.client != nil {
			p.client.Disable = true
			p.client.UnregisterAll()
		}
		if p.server != nil {
			p.server.Disable = true
			p.server.UnregisterAll()
		}

		if p.Client != nil {
			p.Client.Close()
		}
		if p.Server != nil {
			p.Server.Close()
		}
		close(p.toClient)
		close(p.toServer)
	}()
}

// Kick sends attempts to send a player a DisconnectPlayer packet, then quits
// their connection. If packets.NewDisconnectPlayer returns an error, p.Quit is
// called, and the error is returned.
func (p *Player) Kick(msg string) error {
	disc, err := classic.NewDisconnectPlayer(msg)
	if err != nil {
		p.Quit()
		return err
	}
	p.toClient <- disc
	p.Quit()
	return nil
}

// Dial (attempts to) make an outbound connection to the stored hub address.
func (p *Player) Dial() bool {
	server, err := net.Dial("tcp", p.hub)
	if err != nil {
		Infof("(%s) Unable to dial hub: %s", p.Remote(), p.hub)
		Debugf("(%s) Unable to dial remote server: %s (%s)", p.Id, p.hub, err.Error())
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
		if err == nil {
			return
		}

		p.qMutex.Lock()
		defer p.qMutex.Unlock()
		if p.quit || p.quitting {
			return
		}
		panic(err) // Like the mutex matters now..
	}()

	for {
		packet, err := parser.Next()
		if err == ErrParserFinished {
			break
		}

		if err == ErrPacketSkipped {
			continue
		}
		if packet == nil || err != nil {
			Debugf("(%s) readParse(): packet:%+v, err:%#v", p.Id, packet, err)
			p.Quit()
			return
		}

		to <- packet
	}
}

func (p *Player) writeParse(pack <-chan packets.Packet, conn net.Conn) {
	defer func() {
		if err := recover(); !p.quitting && err != nil {
			panic(err)
		}
	}()

	for {
		packet, ok := <-pack
		if packet == nil || !ok {
			Debugf("(%s) writeParse(): nil packet? ok:%v", p.Id, ok)
			p.Quit()
			return
		}

		n, err := conn.Write(packet.Bytes())
		if err != nil {
			Debugf("(%s) writeParse(): conn.Write err: %#v", p.Id, err)
			p.Quit()
			return
		}
		if n != packet.Size() {
			Debugf("(%s) Packet %#.2x is %d bytes, but %d was written", p.Id, packet.Id(), packet.Size(), n)
		}
	}
}

func (p *Player) Parse() {
	// TODO: Dial in a goroutine so that we can parse the client's Identification
	// in the mean time.
	if !p.Dial() {
		return
	}
	Debugf("(%s) Dialed %s!", p.Id, p.Server.RemoteAddr().String())

	t := 2 * time.Second // TODO: Higher, lower? Notchian does 2-3s.
	p.client = NewParser(p, p.Client, packets.ServerBound, t).(*Parser)
	p.server = NewParser(p, p.Server, packets.ClientBound, t).(*Parser)

	// TODO: Config option to log messages?
	//p.client.Register(packets.Message{}, LogMessage)
	//p.server.Register(packets.Message{}, LogMessage)

	// General hooks to drop/debug log packets first.
	//p.client.Register(AllPackets{}, DebugPacket) // TODO
	//p.server.Register(AllPackets{}, DebugPacket) // TODO
	p.client.Register(AllPackets{}, DropPacket)
	p.server.Register(AllPackets{}, DropPacket)

	if p.ku.Config.EdgeCommands {
		p.client.Register(classic.Message{}, EdgeCommand)
	}

	// So we can shove packets down the pipe about identification.
	go p.readParse(p.client, p.toServer)  // C -> B
	go p.writeParse(p.toClient, p.Client) // C <- B

	packet, err := p.client.Next()

	// This might indicate a read timeout, so just in case we shove down a
	// DisconnectPlayer packet and kill their connections.
	if err == ErrParserFinished {
		Infof("(%s) Connected, but didn't send anything in time.", p.Remote())
		p.Kick("You need to log in!")
		return
	}

	if packet == nil || err != nil || packet.Id() != 0x00 {
		// 0x00 = Identification
		Infof("%s didn't identify correctly.", p.Remote())
		Debugf("(%s) !ident: packet:%#v err:%#v", p.Id, packet, err)
		p.Quit()
		return
	}

	// We'll pass it on eventually.
	p.toServer <- packet

	// Store their username!
	var ident *classic.Identification
	ident = packet.(*classic.Identification)
	p.Name = ident.Name
	p.CPE = ident.UserType == 0x42 // Magic value for CPE

	// We handle authentication here if it's enabled, otherwise we just
	// pull out the username from the initial Identification packet.
	// NB: This only supports ClassiCube.
	if p.ku.Config.Authenticate && !compareHash(p.ku.salt, p.Name, ident.KeyMotd) {
		Infof("(%s) Connected, but didn't verify for %s", p.Remote(), p.Name)
		disc, err := classic.NewDisconnectPlayer("Name wasn't verified!")
		if err != nil {
			p.Quit()
			return
		}
		p.toClient <- disc
		p.Quit()
		return
	}

	// Now we can start to pass things along to the server.
	go p.readParse(p.server, p.toClient)  // B <- S
	go p.writeParse(p.toServer, p.Server) // B -> S
}

func NewPlayer(c net.Conn, ku *Kurafuto) (p *Player, err error) {
	p = &Player{
		Id:       uniuri.NewLen(8),
		Client:   c,
		ku:       ku,
		hub:      fmt.Sprintf("%s:%d", ku.Hub.Address, ku.Hub.Port),
		toClient: make(chan packets.Packet, 64),
		toServer: make(chan packets.Packet, 64),

		State: Connecting,

		qMutex: sync.Mutex{},
	}
	return
}
