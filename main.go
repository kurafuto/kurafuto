package main

import (
	"flag"
	"log"
)

var (
	Ku        *Kurafuto
	verbosity int
)

////////////////////

func main() {
	configFile := flag.String("config", "kurafuto.json", "the file your Kurafuto configuration is stored in.")
	forceSalt := flag.String("forceSalt", "", "force a specific salt to be used (don't do this!)")
	flag.IntVar(&verbosity, "v", 0, "Debugging verbosity level.")
	flag.Parse()

	config, err := NewConfigFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	ku, err := NewKurafuto(config)
	if err != nil {
		log.Fatal(err)
	}
	if *forceSalt != "" {
		ku.salt = *forceSalt
	}

	Ku = ku // Make it global.

	Infof("Kurafuto now listening on %s:%d with %d servers", config.Address, config.Port, len(config.Servers))
	Debugf("Debugging level %d enabled! (Salt: %s)", verbosity, Ku.salt)
	if len(config.Ignore) > 0 {
		Debugf("Ignoring these packets: %s", config.Ignore.String())
	}
	if len(config.Drop) > 0 {
		Debugf("Dropping these packets: %s", config.Drop.String())
	}
	if len(config.DropExts) > 0 {
		Debugf("Dropping these extensions: %s", config.DropExts)
	}

	go ku.Run()
	<-ku.Done
}
