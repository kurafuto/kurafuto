package main

import (
	"errors"
	"fmt"
)

type Heartbeat interface {
	Name() string // User displayable heartbeat name, e.g. "ClassiCube"
	Pump() (string, error)
	String() string // String representation of the complete heartbeat URL.
}

// ClassiCube allows Kurafuto to send heartbeats to ClassiCube (classicube.net).
type ClassiCube struct {
	ku *Kurafuto
}

func (h *ClassiCube) Name() string {
	return "ClassiCube"
}

func (h *ClassiCube) Pump() (string, error) {
	// TODO
	return "", errors.New("Not implemented")
}

func (h *ClassiCube) String() string {
	// TODO
	return fmt.Sprintf(`http://www.classicube.net/`)
}

func NewClassiCube(ku *Kurafuto) *ClassiCube {
	return &ClassiCube{ku}
}
