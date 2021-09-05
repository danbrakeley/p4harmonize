package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/danbrakeley/bs"
)

type P4Config struct {
	P4Port   string
	P4User   string
	P4Client string

	name string // display name used in error messages and status reports, set in loadConfig
}

type Config struct {
	Epic  P4Config
	Local P4Config

	// TODO: commit message strings
}

// Load helpers

func loadConfigFromFirstFile(paths []string) (Config, error) {
	for _, path := range paths {
		if bs.IsFile(path) {
			return loadConfigFromFile(path)
		}
	}
	return Config{}, fmt.Errorf("%v do not exist", strings.Join(paths, ", "))
}

func loadConfigFromFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf(`error opening "%s": %w`, path, err)
	}
	defer f.Close()
	return loadConfig(f)
}

func loadConfigFromString(s string) (Config, error) {
	return loadConfig(strings.NewReader(s))
}

func loadConfig(r io.Reader) (Config, error) {
	var cfg Config
	if _, err := toml.NewDecoder(r).Decode(&cfg); err != nil {
		return Config{}, err
	}

	cfg.Epic.name = "epic's server"
	cfg.Local.name = "local server"

	return cfg, nil
}
