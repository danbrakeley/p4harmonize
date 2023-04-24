package main

import (
	"bufio"
	"fmt"
	"io"

	"github.com/danbrakeley/frog"
)

// Logger is a logging interface unique to p4harmonize.
// The intention here is to abstract the underlying logging library.
type Logger interface {
	Src() Logger
	Dst() Logger

	Info(format string, args ...interface{})
	Verbose(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})

	InfoFast(msg string)
	VerboseFast(msg string)
	WarningFast(msg string)
	ErrorFast(msg string)
}

type FrogLog struct {
	Logger  frog.Logger
	Prefix  string
	Palette frog.Palette
}

func MakeLogger(l frog.RootLogger) (log Logger, close func()) {
	l.SetMinLevel(frog.Verbose)
	log = &FrogLog{
		Logger: l,
		Prefix: "",
		Palette: frog.Palette{
			{frog.DarkGray, frog.DarkGray}, // Transient
			{frog.Cyan, frog.DarkGray},     // Verbose
			{frog.White, frog.DarkGray},    // Info
			{frog.Yellow, frog.DarkGray},   // Warning
			{frog.Red, frog.DarkGray},      // Error
		},
	}
	close = func() { l.Close() }
	return log, close
}

func (l *FrogLog) Src() Logger {
	return &FrogLog{
		Logger: l.Logger,
		Prefix: "  >>- ",
		Palette: frog.Palette{
			{frog.DarkGray, frog.DarkBlue}, // Transient
			{frog.DarkBlue, frog.DarkBlue}, // Verbose
			{frog.Blue, frog.DarkBlue},     // Info
			{frog.Yellow, frog.DarkBlue},   // Warning
			{frog.Red, frog.DarkBlue},      // Error
		},
	}
}

func (l *FrogLog) Dst() Logger {
	return &FrogLog{
		Logger: l.Logger,
		Prefix: "  --> ",
		Palette: frog.Palette{
			{frog.DarkGray, frog.DarkGreen},  // Transient
			{frog.DarkGreen, frog.DarkGreen}, // Verbose
			{frog.Green, frog.DarkGreen},     // Info
			{frog.Yellow, frog.DarkGreen},    // Warning
			{frog.Red, frog.DarkGreen},       // Error
		},
	}
}

func (l *FrogLog) logImpl(level frog.Level, format string, args ...interface{}) {
	l.Logger.LogImpl(
		level,
		l.Prefix+fmt.Sprintf(format, args...),
		nil,
		[]frog.PrinterOption{frog.POPalette(l.Palette)},
		frog.ImplData{},
	)
}
func (l *FrogLog) logImplFast(level frog.Level, msg string) {
	l.Logger.LogImpl(
		level,
		l.Prefix+msg,
		nil,
		[]frog.PrinterOption{frog.POPalette(l.Palette)},
		frog.ImplData{},
	)
}

func (l *FrogLog) Info(format string, args ...interface{}) {
	l.logImpl(frog.Info, format, args...)
}
func (l *FrogLog) Verbose(format string, args ...interface{}) {
	l.logImpl(frog.Verbose, format, args...)
}
func (l *FrogLog) Warning(format string, args ...interface{}) {
	l.logImpl(frog.Warning, format, args...)
}
func (l *FrogLog) Error(format string, args ...interface{}) {
	l.logImpl(frog.Error, format, args...)
}

func (l *FrogLog) InfoFast(msg string) {
	l.logImplFast(frog.Info, msg)
}
func (l *FrogLog) VerboseFast(msg string) {
	l.logImplFast(frog.Verbose, msg)
}
func (l *FrogLog) WarningFast(msg string) {
	l.logImplFast(frog.Warning, msg)
}
func (l *FrogLog) ErrorFast(msg string) {
	l.logImplFast(frog.Error, msg)
}

// create an io.Writer that treats each logged line as a call to log.Info

func LogWriter(l Logger) io.Writer {
	r, w := io.Pipe()
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			l.InfoFast(s.Text())
		}
	}()
	return w
}

func LogVerboseWriter(l Logger) io.Writer {
	r, w := io.Pipe()
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			l.VerboseFast(s.Text())
		}
	}()
	return w
}

func LogWarningWriter(l Logger) io.Writer {
	r, w := io.Pipe()
	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			l.WarningFast(s.Text())
		}
	}()
	return w
}
