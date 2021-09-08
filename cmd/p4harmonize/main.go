package main

import (
	"github.com/danbrakeley/bs"
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
	// create a logger and patch bs to use it instead of stdout/stderr
	log := MakeLogger(frog.New(frog.Auto, frog.HideLevel, frog.MessageOnRight, frog.FieldIndent10), "")
	bs.SetColorsEnabled(false)
	bs.SetStdout(LogVerboseWriter(log))
	bs.SetStderr(LogWarningWriter(log))
	bs.SetVerboseEnvVarName("VERBOSE")
	bs.SetVerbose(true)

	cfg, err := loadConfigFromFirstFile(configFileNames)
	if err != nil {
		log.Fatal("Failed to load config: %v", err)
	}

	log.Info("Config loaded from %s", cfg.Filename())

	err = UpdateLocalToMatchEpic(log, cfg)
	if err != nil {
		log.Fatal("%v", err)
	}

	log.Info("Success!")
}
