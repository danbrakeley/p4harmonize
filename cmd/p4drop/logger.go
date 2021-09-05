package main

import "github.com/danbrakeley/bs"

type Logger interface {
	Verbose(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
}

func MakeLogger() Logger {
	return BsLog{}
}

type BsLog struct{}

func (l BsLog) Verbose(format string, args ...interface{}) {
	bs.Verbosef(format, args...)
}

func (l BsLog) Info(format string, args ...interface{}) {
	bs.Echof(format, args...)
}

func (l BsLog) Warn(format string, args ...interface{}) {
	bs.Warnf(format, args...)
}
