package main

import (
	"fmt"
	"os"

	"github.com/danbrakeley/bsh"
	"github.com/proletariatgames/p4harmonize/internal/p4"
)

type srcThreadResults struct {
	Error      error
	ClientRoot string
	Files      []p4.DepotFile
}

func Harmonize(log Logger, cfg Config) error {
	var err error

	// Ensure dst root folder and dst client don't already exist

	if err = preFlightChecks(log, cfg); err != nil {
		return err
	}

	// Start sync from src in the background

	logSrc := log.MakeChildLogger("source")
	chSrc := make(chan srcThreadResults)
	go func() {
		defer close(chSrc)
		chSrc <- srcSyncAndList(logSrc, cfg)
	}()

	// Create dst client

	sh := MakeLoggingBsh(log)
	p4dst := p4.New(sh, cfg.Dst.P4Port, cfg.Dst.P4User, "")

	log.Info("Creating client %s on %s...", cfg.Dst.ClientName, p4dst.DisplayName())

	err = p4dst.CreateClient(cfg.Dst.ClientName, cfg.Dst.ClientRoot, cfg.Dst.ClientStream)
	if err != nil {
		return fmt.Errorf(`Failed to create client %s: %w`, p4dst.Client, err)
	}
	// rebuild p4dst to include the new client and stream
	p4dst = p4.New(sh, cfg.Dst.P4Port, cfg.Dst.P4User, cfg.Dst.ClientName)
	err = p4dst.SetStreamName(cfg.Dst.ClientStream)
	if err != nil {
		return err
	}

	// Force perforce to think you have synced everything already

	log.Info("Slamming %s to head without transferring any files...", cfg.Dst.ClientName)

	err = p4dst.SyncLatestNoDownload()
	if err != nil {
		return err
	}

	// Grab the full list of files

	log.Info("Downloading list of current depot files in destination...")

	dstFiles, err := p4dst.ListDepotFiles()
	if err != nil {
		return fmt.Errorf(`failed to list destination files: %w`, err)
	}

	log.Warning(fmt.Sprintf("P4DST DEPOT FILES: %v", dstFiles))

	// block until sync source sync completes
	str := <-chSrc
	if str.Error != nil {
		return fmt.Errorf("Failed in source thread: %w", str.Error)
	}

	log.Info("Reconciling file lists from source and destination...")

	if !sh.IsDir(str.ClientRoot) {
		return fmt.Errorf("client root %s is missing or is not a folder", str.ClientRoot)
	}

	log.Warning(fmt.Sprintf("SRC ROOT: %v", str.ClientRoot))
	log.Warning(fmt.Sprintf("SRC FILES: %v", str.Files))

	return nil
}

// preFlightChecks performs quick checks to ensure we're in a good state, before
// doing any action that might take a while to complete.
func preFlightChecks(log Logger, cfg Config) error {
	sh := MakeLoggingBsh(log)
	p4dst := p4.New(sh, cfg.Dst.P4Port, cfg.Dst.P4User, "")

	log.Info("Checking folders and clients for %s...", p4dst.DisplayName())

	if sh.Exists(cfg.Dst.ClientRoot) {
		return fmt.Errorf("Destination client root %s already exists. "+
			"Please delete it, or change destination.new_client_root in your config file, then try again.",
			cfg.Dst.ClientRoot)
	}

	clients, err := p4dst.ListClients()
	if err != nil {
		return fmt.Errorf("Failed to get clients from %s: %w", cfg.Dst.P4Port, err)
	}

	hasClient := false
	for _, client := range clients {
		if client == cfg.Dst.ClientName {
			hasClient = true
			break
		}
	}

	if hasClient {
		return fmt.Errorf("Destination client %s already exists on %s. "+
			"Please delete it, or change destination.new_client_name in your config file, then try again.",
			cfg.Dst.ClientName, cfg.Dst.P4Port)
	}

	return nil
}

// srcSyncAndList connects to the source perforce server, syncs to head, then
// requests a list of all file names and types.
func srcSyncAndList(log Logger, cfg Config) srcThreadResults {
	sh := MakeLoggingBsh(log)
	p4src := p4.New(sh, cfg.Src.P4Port, cfg.Src.P4User, cfg.Src.P4Client)

	spec, err := p4src.GetClientSpec()
	if err != nil {
		return srcThreadResults{Error: err}
	}
	root, exists := spec["Root"]
	if !exists {
		return srcThreadResults{Error: fmt.Errorf(`missing field "Root" in client spec "%s"`, p4src.Client)}
	}

	log.Info("Getting latest from %s...", p4src.DisplayName())

	if err := p4src.SyncLatest(); err != nil {
		return srcThreadResults{Error: err}
	}

	log.Info("Downloading list of files for %s on %s...", p4src.Client, p4src.DisplayName())

	files, err := p4src.ListDepotFiles()
	if err != nil {
		return srcThreadResults{Error: fmt.Errorf(`failed to list files from %s: %w`, p4src.DisplayName(), err)}
	}

	return srcThreadResults{
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
