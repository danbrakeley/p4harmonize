package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type SourceConfig struct {
	P4Port   string `toml:"p4port"`
	P4User   string `toml:"p4user"`
	P4Client string `toml:"p4client"`
}

type DestinationConfig struct {
	P4Port       string `toml:"p4port"`
	P4User       string `toml:"p4user"`
	ClientName   string `toml:"new_client_name"`
	ClientRoot   string `toml:"new_client_root"`
	ClientStream string `toml:"new_client_stream"`
}

type Config struct {
	Src SourceConfig      `toml:"source"`
	Dst DestinationConfig `toml:"destination"`

	// TODO: commit message strings?

	// save the file from which this config was loaded, for logging purposes
	filename string
}

func (c *Config) Filename() string {
	return c.filename
}

// Load helpers

func loadConfigFromFirstFile(paths []string) (Config, error) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return loadConfigFromFile(path)
		} else if !os.IsNotExist(err) {
			return Config{}, fmt.Errorf(`error determining if "%s" exists: %v`, path, err)
		}
	}
	return Config{}, fmt.Errorf("%v not found", strings.Join(paths, " or "))
}

func loadConfigFromFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf(`error opening "%s": %w`, path, err)
	}
	defer f.Close()
	cfg, err := loadConfig(f)
	cfg.filename = path
	return cfg, err
}

func loadConfigFromString(s string) (Config, error) {
	return loadConfig(strings.NewReader(s))
}

func loadConfig(r io.Reader) (Config, error) {
	var cfg Config
	if _, err := toml.NewDecoder(r).Decode(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
