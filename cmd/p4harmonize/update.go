package main

import (
	"fmt"
	"os"

	"github.com/danbrakeley/bsh"
	"github.com/proletariatgames/p4harmonize/internal/p4"
)

type epicData struct {
	Error      error
	ClientRoot string
	Files      []p4.DepotFile
}

func UpdateLocalToMatchEpic(log Logger, cfg Config) error {
	var err error

	// Ensure local folder and local client don't already exist

	if err = PreFlightChecks(log, cfg); err != nil {
		return err
	}

	// Start Epic sync in the background

	logEpic := log.MakeChildLogger("epic")
	chEpic := make(chan epicData)
	go func() {
		defer close(chEpic)
		chEpic <- epicSyncAndList(logEpic, cfg)
	}()

	// Create local client

	sh := MakeLoggingBsh(log)
	local := p4.New(sh, cfg.Local.P4Port, cfg.Local.P4User, "")

	log.Info("Creating client %s on %s...", cfg.Local.ClientName, local.DisplayName())

	err = local.CreateClient(cfg.Local.ClientName, cfg.Local.ClientRoot, cfg.Local.ClientStream)
	if err != nil {
		return fmt.Errorf(`Failed to create client %s: %w`, local.Client, err)
	}
	// rebuild the local P4 to include the new client and stream
	local = p4.New(sh, cfg.Local.P4Port, cfg.Local.P4User, cfg.Local.ClientName)
	local.SetStreamName(cfg.Local.ClientStream)

	// Force perforce to think you have synced everything already

	log.Info("Slamming %s to head without transferring any files...", cfg.Local.ClientName)

	err = local.SyncLatestNoDownload()
	if err != nil {
		return err
	}

	// TODO: copy files from EPIC ROOT (ask epic client for the root?) to here

	// block until Epic sync completes
	ed := <-chEpic
	if ed.Error != nil {
		return fmt.Errorf("Failed in Epic thread: %w", ed.Error)
	}
	log.Warning(fmt.Sprintf("EPIC ROOT: %v", ed.ClientRoot))
	log.Warning(fmt.Sprintf("EPIC FILES: %v", ed.Files))

	// Grab the full list from our local server

	log.Info("Downloading list of files for %s on %s...", local.Client, local.DisplayName())

	localFiles, err := local.ListDepotFiles()
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
	sh := MakeLoggingBsh(log)
	local := p4.New(sh, cfg.Local.P4Port, cfg.Local.P4User, "")

	log.Info("Checking folders and clients for %s...", local.DisplayName())

	if sh.Exists(cfg.Local.ClientRoot) {
		return fmt.Errorf("Local client root %s already exists. "+
			"Please delete it, or change local.new_client_root in your config file, then try again.", cfg.Local.ClientRoot)
	}

	clients, err := local.ListClients()
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

// epicSyncAndList connects to the Epic perforce server, syncs to head, then
// requests a list of all file names and types.
func epicSyncAndList(log Logger, cfg Config) epicData {
	sh := MakeLoggingBsh(log)
	epic := p4.New(sh, cfg.Epic.P4Port, cfg.Epic.P4User, cfg.Epic.P4Client)

	spec, err := epic.GetClientSpec()
	if err != nil {
		return epicData{Error: err}
	}
	root, exists := spec["Root"]
	if !exists {
		return epicData{Error: fmt.Errorf(`missing field "Root" in client spec "%s"`, epic.Client)}
	}

	log.Info("Getting latest from %s...", epic.DisplayName())

	if err := epic.SyncLatest(); err != nil {
		return epicData{Error: err}
	}

	log.Info("Downloading list of files for %s on %s...", epic.Client, epic.DisplayName())

	files, err := epic.ListDepotFiles()
	if err != nil {
		return epicData{Error: fmt.Errorf(`failed to list files from %s: %w`, epic.DisplayName(), err)}
	}

	return epicData{
		ClientRoot: root,
		Files:      files,
	}
}

func MakeLoggingBsh(log Logger) *bsh.Bsh {
	w := LogVerboseWriter(log)
	sh := &bsh.Bsh{
		Stdin:        os.Stdin,
		Stdout:       w,
		Stderr:       w,
		DisableColor: true,
	}
	sh.SetVerboseEnvVarName("VERBOSE")
	sh.SetVerbose(true)
	return sh
}
