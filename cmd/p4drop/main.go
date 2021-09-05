package main

import (
	"fmt"
	"os"

	"github.com/danbrakeley/bs"
)

var (
	// config will be loaded from the first extant file from this list
	configFileNames = []string{
		"p4drop.toml",
		"p4drop.tml",
	}
)

func main() {
	bs.SetVerboseEnvVarName("VERBOSE")

	cfg, err := loadConfigFromFirstFile(configFileNames)
	if err != nil {
		bs.Warnf("Failed to load config: %v", err)
		os.Exit(1)
	}

	err = UpdateLocalToMatchEpic(cfg)
	if err != nil {
		bs.Warnf("%v", err)
		os.Exit(1)
	}

	fmt.Println("Success!")
}
