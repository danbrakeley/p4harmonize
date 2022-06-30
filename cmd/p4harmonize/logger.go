package main

import (
	"bufio"
	"fmt"
	"io"

	"github.com/danbrakeley/frog"
)

type Logger interface {
	Close()
	MakeChildLogger(id string) Logger
	Info(format string, args ...interface{})
	Verbose(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})
}

type FrogLog struct {
	Logger frog.Logger
	Fields []frog.Fielder
}

func MakeLogger(l frog.Logger, id string) FrogLog {
	l.SetMinLevel(frog.Verbose)
	return FrogLog{Logger: l}
}

func (l FrogLog) Close() {
	l.Logger.Close()
}

func (l FrogLog) MakeChildLogger(id string) Logger {
	return FrogLog{
		Logger: l.Logger,
		Fields: []frog.Fielder{
			frog.String("thread", id),
		},
	}
}

func (l FrogLog) Info(format string, args ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, args...), l.Fields...)
}

func (l FrogLog) Verbose(format string, args ...interface{}) {
	l.Logger.Verbose(fmt.Sprintf(format, args...), l.Fields...)
}

func (l FrogLog) Warning(format string, args ...interface{}) {
	l.Logger.Warning(fmt.Sprintf(format, args...), l.Fields...)
}

func (l FrogLog) Error(format string, args ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, args...), l.Fields...)
}

// create an io.Writer that treats each logged line as a call to log.Info

func LogWriter(l Logger) io.Writer {
	r, w := io.Pipe()
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			l.Info("%s", s.Text())
		}
	}()
	return w
}

func LogVerboseWriter(l Logger) io.Writer {
	r, w := io.Pipe()
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			l.Verbose("%s", s.Text())
		}
	}()
	return w
}

func LogWarningWriter(l Logger) io.Writer {
	r, w := io.Pipe()
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			l.Warning("%s", s.Text())
		}
	}()
	return w
}
