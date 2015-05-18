package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func sigintQuit(ku *Kurafuto, c <-chan os.Signal) {
	<-c
	if ku == nil {
		// TODO: Log this unfortunate event.
		return
	}
	log.Println("Shutting down!")
	ku.Quit()
}

func sighupReload(ku *Kurafuto, c <-chan os.Signal) {
	for {
		<-c
		if ku == nil {
			// TODO: Log this unfortunate event.
			return
		}
		log.Println("Should reload config now, but that isn't implemented.")
	}
}

////////////////////

// NOTE: Temporary variables, remove them and do dependency injection.
var (
	Ku        *Kurafuto
	Verbosity int
	// Config *Config
)

func main() {
	// Enable Multiple core usage
	runtime.GOMAXPROCS(runtime.NumCPU())

	var (
		configFile = flag.String("config", "kurafuto.json", "the file your Kurafuto configuration is stored in")
		verbosity  = flag.Int("v", 0, "Verbosity level; 0 (default), 1 (info), 2 (debug)")
	)

	flag.Parse()

	config, err := NewConfigFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	ku, err := NewKurafuto(config)
	if err != nil {
		log.Fatal(err)
	}

	Logf("Kurafuto now listening on %s:%d with %d servers", config.Address, config.Port, len(config.Servers))
	Debugf("Debugging level %d enabled! (Salt: %s)", verbosity, ku.salt)

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
	go sigintQuit(ku, sigint)

	sighup := make(chan os.Signal, 3)
	signal.Notify(sighup, syscall.SIGHUP)
	go sighupReload(ku, sighup)

	// The end.
	go ku.Run()
	<-ku.Done
	Log("Bye!")
}
