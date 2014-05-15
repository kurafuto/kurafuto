package main

import (
	"github.com/mgutz/ansi"
	"github.com/sysr-q/kyubu/packets"
	"log"
	"regexp"
)

var (
	resetColor  = ansi.ColorCode("reset")
	colorRegexp = regexp.MustCompile(`&([a-fA-F0-9r])`)
	colors      = map[byte]string{
		'0': ansi.ColorCode("black"),
		'1': ansi.ColorCode("blue"),
		'2': ansi.ColorCode("green"),
		'3': ansi.ColorCode("cyan"),
		'4': ansi.ColorCode("red"),
		'5': ansi.ColorCode("magenta"),
		'6': ansi.ColorCode("yellow"),
		'7': ansi.ColorCode("white"),
		'8': ansi.ColorCode("black+b"),
		'9': ansi.ColorCode("blue+b"),
		'a': ansi.ColorCode("green+b"),
		'b': ansi.ColorCode("cyan+b"),
		'c': ansi.ColorCode("red+b"),
		'd': ansi.ColorCode("magenta+b"),
		'e': ansi.ColorCode("yellow+b"),
		'f': ansi.ColorCode("white+b"),
		// Below are not part of official spec.
		'r': resetColor,
	}
)

// Colorify takes a Minecraft classic chat color-coded string /&[a-f0-9]/, and
// returns a "colorified" string with ANSI escape codes.
func Colorify(in string) string {
	repl := colorRegexp.ReplaceAllFunc([]byte(in), func(s []byte) []byte {
		if len(s) != 2 {
			return s
		}
		b := []byte(colors[s[1]])
		return b
	})
	return string(repl) + resetColor
}

func Debugf(s string, v ...interface{}) {
	if verbosity < 1 {
		return
	}
	log.Printf("[DBUG]"+s, v...)
}

func Infof(s string, v ...interface{}) {
	log.Printf(s, v...)
}

func DebugPacket(id, action string, p packets.Packet) {
	if Ku == nil || Ku.Config == nil {
		return
	}
	for _, id := range Ku.Config.Ignore {
		if id != p.Id() {
			continue
		}
		return
	}
	name := "Unknown"
	if info, ok := packets.Packets[p.Id()]; ok {
		name = info.Name
	}
	Debugf("(%s) %s packet %#.2x (%s)", id, action, p.Id(), name)
}
