package main

import (
	"fmt"
	"os"
	"strings"

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

	diff := Reconcile(str.Files, dstFiles)

	// early out if there's nothing to reconcile
	if len(diff.NearMatch) == 0 && len(diff.SrcOnly) == 0 && len(diff.DstOnly) == 0 {
		log.Info("All files in source and destination already match. Nothing more to do.")
		return nil
	}

	log.Info("Creating changelist in destination...")

	cl, err := p4dst.CreateChangelist("p4harmonize")
	if err != nil {
		return fmt.Errorf("unable to create new changelist: %v", err)
	}

	log.Info("Changelist %d created.", cl)

	for _, pair := range diff.Match {
		log.Warning("EDIT: %s", pair[0].Path)
		// mark file in destination for edit
		// copy file from source root to destination root
	}

	for _, pair := range diff.NearMatch {
		if pair[0].Path != pair[1].Path {
			log.Warning("RENAME: %s to %s", pair[0].Path, pair[1].Path)
		}
		if pair[0].Type != pair[1].Type {
			log.Warning("CHANGE TYPE: %s to %s", pair[0].Path, pair[1].Path)
		}
	}

	for _, src := range diff.SrcOnly {
		log.Warning("ADD: %s", src.Path)
	}

	for _, dst := range diff.DstOnly {
		log.Warning("DELETE: %s", dst.Path)
	}

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

type DepotFileDiff struct {
	Match     [][2]p4.DepotFile // Types and paths are an exact match
	NearMatch [][2]p4.DepotFile // Types and/or path capitalization don't match
	SrcOnly   []p4.DepotFile    // Path only exists in source
	DstOnly   []p4.DepotFile    // Path only exists in destination
}

func Reconcile(src []p4.DepotFile, dst []p4.DepotFile) DepotFileDiff {
	max := len(src)
	if len(dst) > max {
		max = len(dst)
	}

	out := DepotFileDiff{
		Match:     make([][2]p4.DepotFile, 0, max),
		NearMatch: make([][2]p4.DepotFile, 0, max),
		SrcOnly:   make([]p4.DepotFile, 0, max),
		DstOnly:   make([]p4.DepotFile, 0, max),
	}

	is, id := 0, 0
	for is < len(src) && id < len(dst) {
		srcCmp := strings.ToLower(src[is].Path)
		dstCmp := strings.ToLower(dst[id].Path)
		cmp := strings.Compare(srcCmp, dstCmp)
		switch {
		case cmp == 0:
			if src[is].Path == dst[id].Path && src[is].Type == dst[id].Type {
				out.Match = append(out.Match, [2]p4.DepotFile{src[is], dst[id]})
			} else {
				out.NearMatch = append(out.NearMatch, [2]p4.DepotFile{src[is], dst[id]})
			}
			is++
			id++
		case cmp < 0:
			out.SrcOnly = append(out.SrcOnly, src[is])
			is++
		case cmp > 0:
			out.DstOnly = append(out.DstOnly, dst[id])
			id++
		}
	}

	for i := is; i < len(src); i++ {
		out.SrcOnly = append(out.SrcOnly, src[i])
	}

	for i := id; i < len(dst); i++ {
		out.DstOnly = append(out.DstOnly, dst[i])
	}

	return out
}
