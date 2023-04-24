package main

import (
	"fmt"
	"os"

	"github.com/danbrakeley/frog"
)

func createLongtestLogger(filename string) (close func() error, log frog.Logger, err error) {
	// open file
	f, err := os.Create(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("Error creating log file %s: %w", filename, err)
	}

	// console logger
	conlog := frog.New(frog.Auto, frog.POTime(false), frog.POLevel(false), frog.POFieldsLeftMsgRight)

	// file logger
	filelog := frog.NewUnbuffered(f, (&frog.TextPrinter{}).SetOptions(
		frog.POTime(true),
		frog.POFieldsLeftMsgRight,
		frog.POFieldIndent(20),
	))
	filelog.SetMinLevel(frog.Transient)

	// combined logger
	log, teeClose := frog.NewRootTee(conlog, filelog)

	close = func() error {
		teeClose()
		return f.Close()
	}

	return
}
