package main

import (
	"flag"
	"github.com/sysr-q/kyubu/cpe"
	"github.com/sysr-q/kyubu/packets"
	"log"
)

var (
	Ku        *Kurafuto
	verbosity int
)

////////////////////
func Debugf(level int, s string, v ...interface{}) {
	if verbosity < level {
		return
	}
	log.Printf("[DEBUG] "+s, v...)
}

func Packetf(action, id string, p packets.Packet) {
	for _, id := range Ku.Config.Ignore {
		if id != p.Id() {
			continue
		}
		return
	}
	Debugf(1, "[%s] %s Packet %#.2x [%s]", id, action, p.Id(), packets.Packets[p.Id()].Name)
}

func Dropp(p packets.Packet) bool {
	for _, id := range Ku.Config.Drop {
		if id != p.Id() {
			continue
		}
		return true
	}
	if ep, ok := p.(cpe.ExtPacket); ok {
		for _, ext := range Ku.Config.DropExts {
			if ext != ep.String() {
				continue
			}
			return true
		}
	}
	return false
}

func main() {
	var (
		configFile = flag.String("config", "kurafuto.json", "the file your Kurafuto configuration is stored in.")
		forceSalt  = flag.String("forceSalt", "", "force a specific salt to be used (don't do this!)")
	)
	flag.IntVar(&verbosity, "v", 0, "Debugging verbosity level.")
	flag.Parse()

	config, err := NewConfigFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	Ku, err := NewKurafuto(config)
	if err != nil {
		log.Fatal(err)
	}
	if *forceSalt != "" {
		Ku.salt = *forceSalt
	}

	log.Printf("Kurafuto now listening on %s:%d with %d servers", config.Address, config.Port, len(config.Servers))
	Debugf(1, "Debugging level %d enabled! (Salt: %s)", verbosity, Ku.salt)
	Debugf(1, "Ignoring these packets: %s", config.Ignore.String())
	Debugf(1, "Dropping these packets: %s", config.Drop.String())
	Debugf(1, "Dropping these extensions: %s", config.DropExts)

	go Ku.Run()
	<-Ku.Done
}
