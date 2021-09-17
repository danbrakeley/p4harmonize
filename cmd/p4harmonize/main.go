package main

import (
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
	log := MakeLogger(frog.New(frog.Auto, frog.HideLevel, frog.MessageOnRight, frog.FieldIndent10), "")

	cfg, err := loadConfigFromFirstFile(configFileNames)
	if err != nil {
		log.Fatal("Failed to load config: %v", err)
	}

	log.Info("Config loaded from %s", cfg.Filename())

	err = Harmonize(log, cfg)
	if err != nil {
		log.Fatal("%v", err)
	}

	log.Info("Success!")
}
