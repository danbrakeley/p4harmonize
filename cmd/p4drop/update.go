package main

import (
	"fmt"

	"github.com/proletariatgames/p4drop/internal/p4"
)

func UpdateLocalToMatchEpic(log Logger, cfg Config) error {
	epic := p4.New(cfg.Epic.P4Port, cfg.Epic.P4User, cfg.Epic.P4Client)
	local := p4.New(cfg.Local.P4Port, cfg.Local.P4User, cfg.Local.P4Client)

	// Ensure we have nothing checked out in our local depot

	log.Info("Looking for opened files on %s...", local.DisplayName())

	opened, err := local.OpenedFiles()
	if err != nil {
		return err
	}

	if len(opened) > 0 {
		return fmt.Errorf("You have %d file(s) opened on %s. Please revert them before continuing.", len(opened), local.DisplayName())
	}

	// Grab the full list from our local server

	log.Info("Listing all files for %s on %s...", local.Client, local.DisplayName())

	localFiles, err := local.DepotFiles()
	if err != nil {
		return fmt.Errorf(`failed to list files from %s: %w`, cfg.Epic.name, err)
	}

	fmt.Println(localFiles)

	// Grab the full list from Epic

	log.Info("Listing all files for %s on %s...", epic.Client, epic.DisplayName())

	epicFiles, err := epic.DepotFiles()
	if err != nil {
		return fmt.Errorf(`failed to list files from %s: %w`, cfg.Epic.name, err)
	}

	fmt.Println(epicFiles)

	return nil
}
