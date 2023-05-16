package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/danbrakeley/bsh"
	"github.com/danbrakeley/p4harmonize/internal/config"
	"github.com/danbrakeley/p4harmonize/internal/p4"
)

type srcThreadResults struct {
	Success    bool
	ClientRoot string
	Files      []p4.DepotFile
}

func Harmonize(log Logger, cfg config.Config) error {
	var chSrc chan srcThreadResults
	defer func() {
		// if we try to early out before our goroutine is done, then wait for it
		if chSrc != nil {
			<-chSrc
		}
	}()

	var err error

	// Ensure dst root folder and dst client don't already exist

	if !preFlightChecks(log, cfg) {
		return fmt.Errorf("pre-flight checks failed")
	}

	// Start sync from src in a goroutine

	logSrc := log.Src()
	shSrc := MakeLoggingBsh(logSrc)
	chSrc = make(chan srcThreadResults)
	go func() {
		defer close(chSrc)
		chSrc <- srcSyncAndList(logSrc, shSrc, cfg)
	}()

	// Grab dst info and create dst client

	logDst := log.Dst()
	shDst := MakeLoggingBsh(logDst)
	p4dst := p4.New(shDst, cfg.Dst.P4Port, cfg.Dst.P4User, "")

	logDst.Info("Retrieving info for server %s", p4dst.DisplayName())
	info, err := p4dst.Info()
	if err != nil {
		logDst.Error("Failed getting info from server %s: %w", p4dst.DisplayName(), err)
		return fmt.Errorf("error prepping destination server")
	}

	logDst.Info("Creating client %s on %s...", cfg.Dst.ClientName, p4dst.DisplayName())

	err = p4dst.CreateStreamClient(cfg.Dst.ClientName, cfg.Dst.ClientRoot, cfg.Dst.ClientStream)
	if err != nil {
		logDst.Error("Failed to create client %s: %w", p4dst.Client, err)
		return fmt.Errorf("error prepping destination server")
	}
	// set p4dst's client and stream name
	p4dst.Client = cfg.Dst.ClientName
	err = p4dst.SetStreamName(cfg.Dst.ClientStream)
	if err != nil {
		logDst.Error("Unexpected error calling SetStreamName(%s): %v", cfg.Dst.ClientStream, err)
		return fmt.Errorf("error prepping destination server")
	}

	// Force perforce to think you have synced everything already

	logDst.Info("Slamming %s to head without transferring any files...", cfg.Dst.ClientName)
	err = p4dst.SyncLatestNoDownload()
	if err != nil {
		logDst.Error("Failed to update server's view of your local files: %v", err)
		return fmt.Errorf("error prepping destination server")
	}

	// Grab the full list of files

	logDst.Info("Downloading list of current depot files in destination...")
	dstFiles, err := p4dst.ListDepotFiles()
	if err != nil {
		logDst.Error("Failed to list destination files: %v", err)
		return fmt.Errorf("error prepping destination server")
	}

	// block until sync source sync completes
	srcRes := <-chSrc
	chSrc = nil
	if srcRes.Success != true {
		return fmt.Errorf("error syncing from source server")
	}

	if !shSrc.IsDir(srcRes.ClientRoot) {
		logSrc.Error("Client root '%s' is missing or is not a folder", srcRes.ClientRoot)
		return fmt.Errorf("unexpected local file error")
	}

	log.Info("Reconciling file lists from source and destination...")
	var diff DepotFileDiff
	switch info.CaseHandling {
	case p4.CaseInsensitive:
		diff = Reconcile(srcRes.Files, dstFiles, DstIsCaseInsensitive)
	default:
		diff = Reconcile(srcRes.Files, dstFiles)
	}

	// early out if there's nothing to reconcile
	if !diff.HasDifference() {
		log.Info("All files in source and destination already match, so no harmonizing necessary.")
		logDst.Info("Removing unused client...")
		err := p4dst.DeleteClient(p4dst.Client)
		if err != nil {
			logDst.Error("Error deleting client %s: %v", p4dst.Client, err)
			return fmt.Errorf("error cleaning up")
		}
		return nil
	}

	logDst.Info("Creating changelist in destination...")
	cl, err := p4dst.CreateEmptyChangelist("p4harmonize")
	if err != nil {
		logDst.Error("Unable to create new changelist: %v", err)
		return fmt.Errorf("error prepping for changes")
	}

	logDst.Info("Changelist %d created.", cl)
	dstClientRoot, err := filepath.Abs(cfg.Dst.ClientRoot)
	if err != nil {
		logDst.Error("Unable to get absolute path for '%s': %v", cfg.Dst.ClientRoot, err)
		return fmt.Errorf("error prepping for changes")
	}

	var pathsToDelete []string

	// For each file that only exists in the destination, mark it for delete in the destination.
	// NOTE: Process DstOnly BEFORE processing Match, so that any AppleDouble "%" files that
	// got checked directly into the destination are cleaned up properly.
	pathsToDelete = make([]string, 0, len(diff.DstOnly)+len(diff.CaseMismatch))
	for _, dst := range diff.DstOnly {
		dstPath := filepath.Join(dstClientRoot, dst.Path)
		pathsToDelete = append(pathsToDelete, dstPath)
	}
	// NOTE: If the dst server is case insensitive, also delete any files with case mismatches in their paths.
	if len(diff.CaseMismatch) > 0 {
		log.Warning("Files with case problems detected, but the destination server is set to case insensitive mode. " +
			"Perforce cannot fix case issues on a case insensitive server in a single pass. " +
			"When p4harmonize completes, there will be a changelist that deletes files with " +
			"casing issues. After that CL is submitted, please re-run p4harmonize, which will " +
			"re-add the deleted files, but with correct casing." +
			"See https://portal.perforce.com/s/article/3448 for more details.")
		for _, dst := range diff.CaseMismatch {
			dstPath := filepath.Join(dstClientRoot, dst[1].Path)
			pathsToDelete = append(pathsToDelete, dstPath)
		}
	}
	if err := p4dst.Delete(pathsToDelete, p4.Changelist(cl)); err != nil {
		logDst.Error("Unable to mark %d file(s) for delete: %w", len(pathsToDelete), err)
		return fmt.Errorf("error while building changelist")
	}

	// For each file with the capitalization or the types different, copy the file, then make
	// sure perforce is set to fix the mismatch(es).
	matchFilePairsByType := GroupFilePairsByType(diff.Match)

	for newType, diffFiles := range matchFilePairsByType {
		var pathsToEdit []string

		for _, pair := range diffFiles {
			srcPath := filepath.Join(srcRes.ClientRoot, pair[0].Path)
			dstPathNew := filepath.Join(dstClientRoot, pair[0].Path)
			dstPathOld := filepath.Join(dstClientRoot, pair[1].Path)

			// copy file from source root to destination root
			if err := PerforceFileCopy(srcPath, dstPathOld, pair[0].Type); err != nil {
				logDst.Error("%v", err)
				return fmt.Errorf("error while building changelist")
			}

			if dstPathOld != dstPathNew {
				// path has changed, do a single file edit and move
				if err := p4dst.Edit([]string{dstPathOld}, p4.Changelist(cl), p4.Type(newType)); err != nil {
					logDst.Error("Unable to open '%s' for edit: %w", dstPathOld, err)
					return fmt.Errorf("error while building changelist")
				}
				if err := p4dst.Move(dstPathOld, dstPathNew, p4.Changelist(cl), p4.Type(newType)); err != nil {
					logDst.Error("Unable to open '%s' for move to '%s': %v", dstPathOld, dstPathNew, err)
					return fmt.Errorf("error while building changelist")
				}
			} else {
				// add to array for batch edit
				pathsToEdit = append(pathsToEdit, dstPathOld)
			}
		}

		// mark files in destination for edit with type
		if err := p4dst.Edit(pathsToEdit, p4.Changelist(cl), p4.Type(newType)); err != nil {
			logDst.Error("Unable to open %d file(s) for edit: %v", len(pathsToEdit), err)
			return fmt.Errorf("error while building changelist")
		}
	}

	// For each file that only exists in the source, copy it over then add it to the destination.
	srcOnlyFilesByType := GroupFilesByType(diff.SrcOnly)

	for srcType, srcFiles := range srcOnlyFilesByType {
		var pathsToAdd []string

		for _, src := range srcFiles {
			srcPath := filepath.Join(srcRes.ClientRoot, src.Path)
			dstPath := filepath.Join(dstClientRoot, src.Path)

			// copy file from source root to destination root
			if err := PerforceFileCopy(srcPath, dstPath, src.Type); err != nil {
				logDst.Error("%v", err)
				return fmt.Errorf("error while building changelist")
			}

			// add to the depot
			dstPathForAdd, err := p4.UnescapePath(dstPath)
			if err != nil {
				logDst.Error("Error unescaping '%s': %w", dstPath, err)
				return fmt.Errorf("error while building changelist")
			}

			pathsToAdd = append(pathsToAdd, dstPathForAdd)
		}

		if err := p4dst.Add(pathsToAdd, p4.Changelist(cl), p4.Type(srcType), p4.DoNotIgnore); err != nil {
			logDst.Error("Unable to open %d file(s) for add: %w", len(pathsToAdd), err)
			return fmt.Errorf("error while building changelist")
		}
	}

	// TODO: Do we ALWAYS need to do this? There is a note in the digest code that suggests that
	// sometimes digests may not be available, in which case this revert is necessary.
	// Is that the only case? If so, can we explicitly detect that, and only do this in that case?
	if err := p4dst.RevertUnchanged(filepath.Join(dstClientRoot, "..."), p4.Changelist(cl)); err != nil {
		logDst.Error("Unable to revert unchanged files in the destination: %w", err)
		return fmt.Errorf("error while building changelist")
	}

	root, err := filepath.Abs(cfg.Dst.ClientRoot)
	if err != nil {
		root = cfg.Dst.ClientRoot
	}

	log.Warning("Success! All changes are waiting in CL #%d. Please review and submit when ready.", cl)

	if len(diff.CaseMismatch) > 0 {
		log.Error("Due to file casing problems, you will need to re-run p4harmonize after submitting the above CL.")
		log.Error("See https://portal.perforce.com/s/article/3448 for more details.")
	}

	log.Info("Remember to delete workspace \"%s\"", cfg.Dst.ClientName)
	log.Info("and local folder \"%s\"", root)

	return nil
}

// preFlightChecks performs quick checks to ensure we're in a good state, before
// doing any action that might take a while to complete.
func preFlightChecks(log Logger, cfg config.Config) bool {

	// verify we're logged in to both src and dst

	logSrc := log.Src()
	shSrc := MakeLoggingBsh(logSrc)
	p4src := p4.New(shSrc, cfg.Src.P4Port, cfg.Src.P4User, "")

	if needsLogin, err := p4src.NeedsLogin(); err != nil {
		logSrc.Error("Error checking login status on %s: %v", p4src.Port, err)
		return false
	} else if needsLogin {
		logSrc.Error("Not logged in. Please run 'p4 -p %s -u %s login' and then try again.", p4src.Port, p4src.User)
		return false
	}

	logDst := log.Dst()
	shDst := MakeLoggingBsh(logDst)
	p4dst := p4.New(shDst, cfg.Dst.P4Port, cfg.Dst.P4User, "")

	if needsLogin, err := p4dst.NeedsLogin(); err != nil {
		logDst.Error("Error checking login status on %s: %v", p4dst.Port, err)
		return false
	} else if needsLogin {
		logDst.Error("Not logged in. Please run 'p4 -p %s -u %s login' and then try again.", p4dst.Port, p4dst.User)
		return false
	}

	// verify destination folders and clients we want to create don't already exist

	if shDst.Exists(cfg.Dst.ClientRoot) {
		logDst.Error("Destination client root '%s' already exists.", cfg.Dst.ClientRoot)
		logDst.Error("Please delete it, or change `destination.new_client_root` in your config file, then try again.")
		return false
	}

	clients, err := p4dst.ListClients()
	if err != nil {
		logDst.Error("Failed to get clients from %s: %v", cfg.Dst.P4Port, err)
		return false
	}

	hasClient := false
	for _, client := range clients {
		if client == cfg.Dst.ClientName {
			hasClient = true
			break
		}
	}

	if hasClient {
		logDst.Error("Destination client %s already exists on %s.", cfg.Dst.ClientName, cfg.Dst.P4Port)
		logDst.Error("Please delete it, or change `destination.new_client_name` in your config file, then try again.")
		return false
	}

	return true
}

// srcSyncAndList connects to the source perforce server, syncs to head, then
// requests a list of all file names and types.
func srcSyncAndList(logSrc Logger, shSrc *bsh.Bsh, cfg config.Config) srcThreadResults {
	p4src := p4.New(shSrc, cfg.Src.P4Port, cfg.Src.P4User, cfg.Src.P4Client)

	spec, err := p4src.GetClientSpec()
	if err != nil {
		logSrc.Error("Failed to get client spec: %v", err)
		return srcThreadResults{Success: false}
	}
	root, exists := spec["Root"]
	if !exists {
		logSrc.Error("Missing field `Root` in client spec %s", p4src.Client)
		return srcThreadResults{Success: false}
	}

	logSrc.Info("Getting latest from source...")

	if err := p4src.SyncLatest(); err != nil {
		logSrc.Error("Failed to sync latest: %v", err)
		return srcThreadResults{Success: false}
	}

	logSrc.Info("Downloading list of files with types from source...")

	files, err := p4src.ListDepotFiles()
	if err != nil {
		logSrc.Error("Failed to list files from source: %v", err)
		return srcThreadResults{Success: false}
	}

	return srcThreadResults{
		Success:    true,
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
	Match   [][2]p4.DepotFile // Paths match, but type, case, or content may not (see CaseMismatch below for exceptions).
	SrcOnly []p4.DepotFile    // Path only exists in source
	DstOnly []p4.DepotFile    // Path only exists in destination

	// When the dst server is in case insensitive mode, any case mismatches must be handled specially.
	// In this case, Match will not list files that have case mismatches in their path, and instead
	// those files will only be listed here in CaseMismatch.
	CaseMismatch [][2]p4.DepotFile
}

// HasDifference returns true if this struct contains any differences at all
func (d *DepotFileDiff) HasDifference() bool {
	return len(d.Match) > 0 || len(d.SrcOnly) > 0 || len(d.DstOnly) > 0 || len(d.CaseMismatch) > 0
}

type ReconcileOption uint8

const (
	DstIsCaseInsensitive ReconcileOption = iota // case insensitive dst servers need special handling of casing issues
)

func Reconcile(src []p4.DepotFile, dst []p4.DepotFile, opts ...ReconcileOption) DepotFileDiff {
	// parse options
	dstIsCaseInsensitive := false
	for _, v := range opts {
		if v == DstIsCaseInsensitive {
			dstIsCaseInsensitive = true
		}
	}

	max := len(src)
	if len(dst) > max {
		max = len(dst)
	}

	out := DepotFileDiff{
		Match:   make([][2]p4.DepotFile, 0, max),
		SrcOnly: make([]p4.DepotFile, 0, max),
		DstOnly: make([]p4.DepotFile, 0, max),
	}

	if dstIsCaseInsensitive {
		out.CaseMismatch = make([][2]p4.DepotFile, 0, max)
	}

	is, id := 0, 0
	for is < len(src) && id < len(dst) {
		srcCmp := strings.ToLower(src[is].Path)
		dstCmp := strings.ToLower(dst[id].Path)
		cmp := strings.Compare(srcCmp, dstCmp)
		switch {
		case cmp == 0:
			caseDifference := src[is].Path != dst[id].Path
			// A dst server that is case insensitive will need special handling to fix case issues.
			if dstIsCaseInsensitive && caseDifference {
				out.CaseMismatch = append(out.CaseMismatch, [2]p4.DepotFile{src[is], dst[id]})
				// Case mismatch will be handled by deleting the dst file, and requesting the user to
				// run a second pass to get it re-added with the proper case.
				// So we don't need to know anything else about this file right now.
			} else {
				typeDifference := src[is].Type != dst[id].Type
				// If we don't have a digest, assume it's different (since we'll use `p4 revert -a` to
				// cleanup at the end), otherwise compare digests to see if there's a difference.
				contentDifference := len(src[is].Digest) == 0 || src[is].Digest != dst[id].Digest
				if caseDifference || typeDifference || contentDifference {
					out.Match = append(out.Match, [2]p4.DepotFile{src[is], dst[id]})
				}
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

	if is < len(src) {
		out.SrcOnly = append(out.SrcOnly, src[is:]...)
	}

	if id < len(dst) {
		out.DstOnly = append(out.DstOnly, dst[id:]...)
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

// Splits a list of file pairs (new, old) into groups in which each group
// shares the same perforce file type (for the new file)
func GroupFilePairsByType(filePairs [][2]p4.DepotFile) map[string][][2]p4.DepotFile {
	filePairsByType := map[string][][2]p4.DepotFile{}

	for _, filePair := range filePairs {
		filePairsByType[filePair[0].Type] = append(filePairsByType[filePair[0].Type], filePair)
	}

	return filePairsByType
}
