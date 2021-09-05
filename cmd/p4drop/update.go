package main

import (
	"fmt"

	"github.com/proletariatgames/p4drop/internal/p4"
)

func UpdateLocalToMatchEpic(cfg Config) error {
	epic := p4.New(cfg.Epic.P4Port, cfg.Epic.P4User, cfg.Epic.P4Client)

	epicFiles, err := epic.ListFiles()
	if err != nil {
		return fmt.Errorf(`failed to list files from %s: %w`, cfg.Epic.name, err)
	}

	fmt.Println(epicFiles)

	return nil
}
