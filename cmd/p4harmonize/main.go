package main

import (
	"os"
	"time"

	"github.com/danbrakeley/frog"
)

var (
	// config will be loaded from the first extant file from this list
	configFileNames = []string{
		"config.toml",
		"config.tml",
	}
)

func main() {
	status := mainExit()
	if status != 0 {
		// From os/proc.go: "For portability, the status code should be in the range [0, 125]."
		if status < 0 || status > 125 {
			status = 125
		}
		os.Exit(status)
	}
}

func mainExit() int {
	start := time.Now()
	log := MakeLogger(frog.New(frog.Auto, frog.HideLevel, frog.MessageOnRight, frog.FieldIndent10), "")
	defer func() {
		dur := time.Now().Sub(start)
		log.Info("Running Time: %v", dur)
		log.Logger.SetMinLevel(frog.Info)
		log.Close()
	}()

	cfg, err := loadConfigFromFirstFile(configFileNames)
	if err != nil {
		log.Error("Failed to load config: %v", err)
		return 1
	}

	log.Info("Config loaded from %s", cfg.Filename())

	err = Harmonize(log, cfg)
	if err != nil {
		log.Error("%v", err)
		return 2
	}

	log.Info("Success!")
	return 0
}
