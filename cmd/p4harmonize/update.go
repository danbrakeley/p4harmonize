package main

import (
	"fmt"

	"github.com/danbrakeley/bs"
	"github.com/proletariatgames/p4harmonize/internal/p4"
)

func UpdateLocalToMatchEpic(log Logger, cfg Config) error {
	var err error

	// Ensure local folder and local client don't already exist
	if err = PreFlightChecks(log, cfg); err != nil {
		return err
	}

	local := p4.New(cfg.Local.P4Port, cfg.Local.P4User, "")

	// Start Epic sync in the background

	var epicFiles []p4.DepotFile
	logEpic := log.MakeChildLogger("epic")
	chEpic := make(chan error)
	go func() {
		defer close(chEpic)
		chEpic <- EpicSyncAndList(logEpic, cfg, &epicFiles)
	}()

	// Create local client

	log.Info("Creating client %s on %s...", cfg.Local.ClientName, local.DisplayName())

	err = local.CreateClient(LogVerboseWriter(log), LogWarningWriter(log), cfg.Local.ClientName, cfg.Local.ClientRoot, cfg.Local.ClientStream)
	if err != nil {
		return fmt.Errorf(`Failed to create client %s: %w`, local.Client, err)
	}
	// rebuild the local P4 to include the new client and stream
	local = p4.New(cfg.Local.P4Port, cfg.Local.P4User, cfg.Local.ClientName)
	local.SetStreamName(cfg.Local.ClientStream)

	// Force perforce to think you have synced everything already

	log.Info("Slamming %s to head without actually syncing any files...", cfg.Local.ClientName)

	err = local.FakeLatest(LogWarningWriter(log))
	if err != nil {
		return err
	}

	// TODO: copy files from EPIC ROOT (ask epic client for the root?) to here

	// block until Epic sync completes
	err = <-chEpic
	if err != nil {
		return fmt.Errorf("Failed in Epic thread: %w", err)
	}
	log.Warning(fmt.Sprintf("EPIC FILES: %v", epicFiles))

	// Grab the full list from our local server

	log.Info("Downloading list of files for %s on %s...", local.Client, local.DisplayName())

	localFiles, err := local.DepotFiles()
	if err != nil {
		return fmt.Errorf(`failed to list files from %s: %w`, local.DisplayName(), err)
	}

	log.Warning(fmt.Sprintf("LOCAL FILES: %v", localFiles))

	return nil
}

// PreFlightChecks performs quick checks to ensure we're in a good state, before
// involving the Epic perforce server (which can be slow to respond), and before
// doing any action that might take a while to complete.
func PreFlightChecks(log Logger, cfg Config) error {
	local := p4.New(cfg.Local.P4Port, cfg.Local.P4User, "")

	log.Info("Checking folders and clients for %s...", local.DisplayName())

	if bs.Exists(cfg.Local.ClientRoot) {
		return fmt.Errorf("Local client root %s already exists. "+
			"Please delete it, or change local.new_client_root in your config file, then try again.", cfg.Local.ClientRoot)
	}

	clients, err := local.Clients()
	if err != nil {
		return fmt.Errorf("Failed to get clients from %s: %w", cfg.Local.P4Port, err)
	}

	hasClient := false
	for _, client := range clients {
		if client == cfg.Local.ClientName {
			hasClient = true
			break
		}
	}

	if hasClient {
		return fmt.Errorf("Local client %s already exists on %s. "+
			"Please delete it, or change local.new_client_name in your config file, then try again.",
			cfg.Local.ClientName, cfg.Local.P4Port)
	}

	return nil
}

// EpicSyncAndList connects to the Epic perforce server, syncs to head, then
// requests a list of all file names and types.
func EpicSyncAndList(log Logger, cfg Config, epicFiles *[]p4.DepotFile) error {
	epic := p4.New(cfg.Epic.P4Port, cfg.Epic.P4User, cfg.Epic.P4Client)

	log.Info("Getting latest from %s...", epic.DisplayName())

	err := epic.GetLatest(LogVerboseWriter(log), LogWarningWriter(log))
	if err != nil {
		return err
	}

	log.Info("Downloading list of files for %s on %s...", epic.Client, epic.DisplayName())

	files, err := epic.DepotFiles()
	if err != nil {
		return fmt.Errorf(`failed to list files from %s: %w`, epic.DisplayName(), err)
	}

	*epicFiles = files
	return nil
}
