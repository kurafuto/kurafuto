package main

import (
	"github.com/mgutz/ansi"
	"log"
	"regexp"
)

var (
	resetColor = ansi.ColorCode("reset")
	fatalColor = ansi.ColorCode("red+b")
	warnColor  = ansi.ColorCode("yellow+b")
	debugColor = ansi.ColorCode("blue")
	infoColor  = ansi.ColorCode("blue+b")
)

var (
	colorRegexp = regexp.MustCompile(`&([a-fA-F0-9r])`)
	colors      = map[byte]string{
		// TODO: the rest of the color codes.
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

func Fatalf(s string, v ...interface{}) {
	log.Fatalf(fatalColor+s+resetColor, v...)
}

func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

func Warnf(s string, v ...interface{}) {
	log.Printf(warnColor+s+resetColor, v...)
}

func Debugf(s string, v ...interface{}) {
	if Verbosity < 2 {
		return
	}
	log.Printf(debugColor+s+resetColor, v...)
}

func Infof(s string, v ...interface{}) {
	if Verbosity < 1 {
		return
	}
	log.Printf(infoColor+s+resetColor, v...)
}

func Logf(s string, v ...interface{}) {
	log.Printf(s, v...)
}

func Log(s ...interface{}) {
	log.Println(s...)
}
