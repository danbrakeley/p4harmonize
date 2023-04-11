package main

import (
	"fmt"
)

type ServerType uint8

const (
	Src ServerType = iota
	Dst ServerType = iota
)

type Server struct {
	t      ServerType
	port   string
	user   string
	depot  string
	stream string
	root   string
}

func (s *Server) IsSrc() bool {
	return s.t == Src
}

func (s *Server) IsDst() bool {
	return s.t == Dst
}

func (s *Server) Port() string {
	return s.port
}

func (s *Server) User() string {
	return s.user
}

// depot name, e.g. "UE4"
func (s *Server) Depot() string {
	return s.depot
}

// stream name, e.g. "Release-4.20"
func (s *Server) StreamName() string {
	return s.stream
}

// fulll stream path, e.g. "//UE4/Release-4.20"
func (s *Server) StreamPath() string {
	return fmt.Sprintf("//%s/%s", s.depot, s.stream)
}

func (s *Server) Root() string {
	return s.root
}

func (s *Server) Client() string {
	return fmt.Sprintf("%s-%s-%s", s.user, s.depot, s.stream)
}
