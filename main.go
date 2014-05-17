package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	Ku        *Kurafuto
	verbosity int
)

func sigintQuit(c <-chan os.Signal) {
	<-c
	if Ku == nil {
		return
	}
	log.Println("Shutting down!")
	Ku.Quit()
}

func sighupReload(c <-chan os.Signal) {
	for {
		<-c
		if Ku == nil {
			return
		}
		log.Println("Should reload config now, but that isn't implemented.")
	}
}

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

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	go sigintQuit(sigint)

	sighup := make(chan os.Signal, 3)
	signal.Notify(sighup, syscall.SIGHUP)
	go sighupReload(sighup)

	go ku.Run()
	<-ku.Done
}
