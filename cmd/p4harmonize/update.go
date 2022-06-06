package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		return fmt.Errorf("Failed to create client %s: %w", p4dst.Client, err)
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
		return fmt.Errorf("Failed to list destination files: %w", err)
	}

	// block until sync source sync completes
	str := <-chSrc
	if str.Error != nil {
		return fmt.Errorf("Failed in source thread: %w", str.Error)
	}

	log.Info("Reconciling file lists from source and destination...")

	if !sh.IsDir(str.ClientRoot) {
		return fmt.Errorf("Client root '%s' is missing or is not a folder", str.ClientRoot)
	}

	diff := Reconcile(str.Files, dstFiles)

	// early out if there's nothing to reconcile
	if !diff.HasDifference {
		log.Info("All files in source and destination already match, so no harmonizing necessary.")
		log.Info("Removing unused client...")
		err := p4dst.DeleteClient(p4dst.Client)
		if err != nil {
			return fmt.Errorf("Error deleting client %s: %w", p4dst.Client, err)
		}
		return nil
	}

	log.Info("Creating changelist in destination...")

	cl, err := p4dst.CreateChangelist("p4harmonize")
	if err != nil {
		return fmt.Errorf("Unable to create new changelist: %v", err)
	}

	log.Info("Changelist %d created.", cl)

	dstClientRoot, err := filepath.Abs(cfg.Dst.ClientRoot)
	if err != nil {
		return fmt.Errorf("Unable to get absolute path for '%s': %w", cfg.Dst.ClientRoot, err)
	}

	// For each file that only exists in the destination, mark it for delete in the destination.
	// NOTE: Process DstOnly BEFORE processing Match, so that any AppleDouble "%" files that
	// got checked directly into the destination are cleaned up properly.
	for _, dst := range diff.DstOnly {
		dstPath := filepath.Join(dstClientRoot, dst.Path)
		if err := p4dst.Delete(dstPath, p4.Changelist(cl)); err != nil {
			return fmt.Errorf("Unable to mark '%s' for delete: %w", dstPath, err)
		}
	}

	// For each file with the capitalization or the types different, copy the file, then make
	// sure perforce is set to fix the mismatch(es).
	for _, pair := range diff.Match {
		srcPath := filepath.Join(str.ClientRoot, pair[0].Path)
		dstPathOld := filepath.Join(dstClientRoot, pair[1].Path)
		dstPathNew := filepath.Join(dstClientRoot, pair[0].Path)

		// copy file from source root to destination root
		if err := PerforceFileCopy(srcPath, dstPathOld, pair[0].Type); err != nil {
			return err
		}

		// mark file in destination for edit with type
		if err := p4dst.Edit(dstPathOld, p4.Changelist(cl), p4.Type(pair[0].Type)); err != nil {
			return fmt.Errorf("Unable to open '%s' for edit: %w", dstPathOld, err)
		}

		if dstPathOld != dstPathNew {
			// mark file in destination for move
			if err := p4dst.Move(dstPathOld, dstPathNew, p4.Changelist(cl), p4.Type(pair[0].Type)); err != nil {
				return fmt.Errorf("Unable to open '%s' for move to '%s': %w", dstPathOld, dstPathNew, err)
			}
		}
	}

	// For each file that only exists in the source, copy it over then add it to the destination.
	srcOnlyFilesByType := GroupFilesByType(diff.SrcOnly)

	for srcType, srcFiles := range srcOnlyFilesByType {
		var pathsToAdd []string

		for _, src := range srcFiles {
			srcPath := filepath.Join(str.ClientRoot, src.Path)
			dstPath := filepath.Join(dstClientRoot, src.Path)

			// copy file from source root to destination root
			if err := PerforceFileCopy(srcPath, dstPath, src.Type); err != nil {
				return err
			}

			// add to the depot
			dstPathForAdd, err := p4.UnescapePath(dstPath)
			if err != nil {
				return fmt.Errorf("Error unescaping '%s': %w", dstPath, err)
			}

			pathsToAdd = append(pathsToAdd, dstPathForAdd)
		}

		if err := p4dst.Add(pathsToAdd, p4.Changelist(cl), p4.Type(srcType)); err != nil {
			return fmt.Errorf("Unable to open %d file(s) for add: %w", len(pathsToAdd), err)
		}
	}

	if err := p4dst.RevertUnchanged(filepath.Join(dstClientRoot, "..."), p4.Changelist(cl)); err != nil {
		return fmt.Errorf("Unable to revert unchanged files in the destination: %w", err)
	}

	log.Warning("Success! All changes are waiting in CL #%d. Please review and submit when ready.", cl)

	return nil
}

// preFlightChecks performs quick checks to ensure we're in a good state, before
// doing any action that might take a while to complete.
func preFlightChecks(log Logger, cfg Config) error {
	sh := MakeLoggingBsh(log)

	log.Info("Checking login ticket status...")

	p4src := p4.New(sh, cfg.Src.P4Port, cfg.Src.P4User, "")

	if needsLogin, err := p4src.NeedsLogin(); err != nil {
		return fmt.Errorf("Error checking login status on %s: %w", p4src.Port, err)
	} else if needsLogin {
		return fmt.Errorf("Not logged in. Please run 'p4 -p %s -u %s login' and then try again.", p4src.Port, p4src.User)
	}

	p4dst := p4.New(sh, cfg.Dst.P4Port, cfg.Dst.P4User, "")

	if needsLogin, err := p4dst.NeedsLogin(); err != nil {
		return fmt.Errorf("Error checking login status on %s: %w", p4dst.Port, err)
	} else if needsLogin {
		return fmt.Errorf("Not logged in. Please run 'p4 -p %s -u %s login' and then try again.", p4dst.Port, p4dst.User)
	}

	log.Info("Checking if destination folder or client already exist...")

	if sh.Exists(cfg.Dst.ClientRoot) {
		return fmt.Errorf("Destination client root '%s' already exists. "+
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
		return srcThreadResults{Error: fmt.Errorf("Missing field Root in client spec %s", p4src.Client)}
	}

	log.Info("Getting latest from source...")

	if err := p4src.SyncLatest(); err != nil {
		return srcThreadResults{Error: err}
	}

	log.Info("Downloading list of files with types from source...")

	files, err := p4src.ListDepotFiles()
	if err != nil {
		return srcThreadResults{Error: fmt.Errorf("Failed to list files from source: %w", err)}
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
	HasDifference bool
	Match         [][2]p4.DepotFile // Types and paths are an exact match
	SrcOnly       []p4.DepotFile    // Path only exists in source
	DstOnly       []p4.DepotFile    // Path only exists in destination
}

func Reconcile(src []p4.DepotFile, dst []p4.DepotFile) DepotFileDiff {
	max := len(src)
	if len(dst) > max {
		max = len(dst)
	}

	out := DepotFileDiff{
		Match:   make([][2]p4.DepotFile, 0, max),
		SrcOnly: make([]p4.DepotFile, 0, max),
		DstOnly: make([]p4.DepotFile, 0, max),
	}

	is, id := 0, 0
	for is < len(src) && id < len(dst) {
		srcCmp := strings.ToLower(src[is].Path)
		dstCmp := strings.ToLower(dst[id].Path)
		cmp := strings.Compare(srcCmp, dstCmp)
		switch {
		case cmp == 0:
			caseDifference := src[is].Path != dst[id].Path
			typeDifference := src[is].Type != dst[id].Type
			// If we don't have a digest, assume it's different (since we'll use `p4 revert -a`),
			// otherwise compare digests to see if there's a difference.
			contentDifference := len(src[is].Digest) == 0 || src[is].Digest != dst[id].Digest
			if caseDifference || typeDifference || contentDifference {
				out.Match = append(out.Match, [2]p4.DepotFile{src[is], dst[id]})
				out.HasDifference = true
			}
			is++
			id++
		case cmp < 0:
			out.SrcOnly = append(out.SrcOnly, src[is])
			out.HasDifference = true
			is++
		case cmp > 0:
			out.DstOnly = append(out.DstOnly, dst[id])
			out.HasDifference = true
			id++
		}
	}

	for i := is; i < len(src); i++ {
		out.SrcOnly = append(out.SrcOnly, src[i])
		out.HasDifference = true
	}

	for i := id; i < len(dst); i++ {
		out.DstOnly = append(out.DstOnly, dst[i])
		out.HasDifference = true
	}

	return out
}

// PerforceFileCopy copies file "src" to file/path "dst", creating any missing directories needed by "dst",
// and handling Perforce escape characters (%00) properly.
func PerforceFileCopy(src, dst, filetype string) error {
	srcPath, err := p4.UnescapePath(src)
	if err != nil {
		return err
	}
	dstPath, err := p4.UnescapePath(dst)
	if err != nil {
		return err
	}

	switch filetype {
	case "apple":
		srcDouble := filepath.Join(filepath.Dir(srcPath), "%"+filepath.Base(srcPath))
		dstDouble := filepath.Join(filepath.Dir(dstPath), "%"+filepath.Base(dstPath))
		if err := verifyAndCopy(srcDouble, dstDouble); err != nil {
			return err
		}
	}

	return verifyAndCopy(srcPath, dstPath)
}

func verifyAndCopy(srcPath, dstPath string) error {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("Unable to stat '%s': %w", srcPath, err)
	}

	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("'%s' is not a regular file", srcPath)
	}
	srcSize := srcInfo.Size()

	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		return fmt.Errorf("Unable to mkdir '%s': %w", dstDir, err)
	}

	s, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("Unable to open '%s': %w", srcPath, err)
	}
	defer s.Close()

	d, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("Unable to create '%s': %w", dstPath, err)
	}
	defer d.Close()
	n, err := io.Copy(d, s)
	if err != nil {
		return fmt.Errorf("Unable to copy '%s' to '%s': %w", srcPath, dstPath, err)
	}
	if n != srcSize {
		return fmt.Errorf("Expected '%s' to copy %d bytes to '%s', but only %d were copied", srcPath, n, dstPath, srcSize)
	}
	return nil
}

// Splits a list of files into groups of files in which each group
// shares the same perforce file type
func GroupFilesByType(files []p4.DepotFile) map[string][]p4.DepotFile {
	filesByType := map[string][]p4.DepotFile{}

	for _, file := range files {
		filesByType[file.Type] = append(filesByType[file.Type], file)
	}

	return filesByType
}
