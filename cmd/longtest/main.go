package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/danbrakeley/bsh"
	"github.com/danbrakeley/frog"
	"github.com/danbrakeley/p4harmonize/internal/config"
	"github.com/danbrakeley/p4harmonize/internal/p4"
)

var Servers = []Server{
	{Src, "1661", "super", "UE4", "Release-4.20", "./p4/1"},
	{Src, "1662", "super", "UE4", "Release-4.20", "./p4/2"},
	{Dst, "1663", "super", "test", "engine", "./p4/3"},
	{Dst, "1664", "super", "test", "engine", "./p4/4"},
}

func main() {
	status := mainExit()
	if status != 0 {
		// From os/proc.go: "For portability, the status code should be in the range [0, 125]."
		if status < 0 || status > 125 {
			status = 125
		}
		os.Exit(status)
	}
}

func mainExit() int {
	close, log, err := createLongtestLogger("longtest.log")
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer close()

	// SETUP

	log.Warning(fmt.Sprintf("Populating %d perforce servers with test data in parallel...", len(Servers)))
	bringUpServers(log, Servers)

	// RUN

	log.Warning("Running tests in parallel...")

	chErr := make(chan error)

	go func() { chErr <- runTwoServers(log, 1663, Servers[0], Servers[2], 3, 1) }()
	go func() { chErr <- runTwoServers(log, 1664, Servers[1], Servers[3], 3, 2) }()

	<-chErr
	<-chErr

	return 0
}

// connect a frog logger to a bsh shell, piping all output from bsh into the logger
func createBshWithTransientLogger(log frog.Logger, fields ...frog.Fielder) (close func(), sh *bsh.Bsh) {
	rout, wout := io.Pipe()
	rerr, werr := io.Pipe()
	close = func() {
		if err := werr.Close(); err != nil {
			log.Error("error closing bsh err writer", frog.Err(err))
		}
		if err := wout.Close(); err != nil {
			log.Error("error closing bsh out writer", frog.Err(err))
		}
	}

	// Start a goroutine that reads lines off stdout
	go func() {
		scanner := bufio.NewScanner(rout)
		for scanner.Scan() {
			line := scanner.Text()
			log.Transient(line, fields...)
		}

		err := scanner.Err()
		if err != nil {
			log.Error("error in transient scanner", frog.Err(err))
		}
	}()

	// Start a goroutine that reads lines off stderr
	go func() {
		scanner := bufio.NewScanner(rerr)
		for scanner.Scan() {
			line := scanner.Text()
			log.Error(line, fields...)
		}

		err := scanner.Err()
		if err != nil {
			log.Error("error in error scanner", frog.Err(err))
		}
	}()

	sh = &bsh.Bsh{
		Stdout:       wout,
		Stderr:       werr,
		DisableColor: true,
	}

	return
}

func bringUpServers(logParent frog.Logger, Servers []Server) {
	var wg sync.WaitGroup
	wg.Add(len(Servers))

	for _, v := range Servers {
		l := frog.AddAnchor(logParent)
		defer frog.RemoveAnchor(l)

		go func(log frog.Logger, s Server) {
			log = frog.WithFields(log, frog.String("stage", "init"), frog.String("p4port", s.Port()))
			defer wg.Done()
			close, sh := createBshWithTransientLogger(log)
			defer close()
			pf := p4.New(sh, s.Port(), s.User(), "")
			if s.IsSrc() {
				if err := setupSrc(sh, pf, s); err != nil {
					log.Error("error in setupSrc", frog.Err(err))
					return
				}
			} else {
				if err := setupDst(sh, pf, s); err != nil {
					log.Error("error in setupSrc", frog.Err(err))
					return
				}
			}
			log.Info("Server ready")
		}(l, v)
	}

	wg.Wait()
}

func writeConfig(file string, src Server, dst Server) error {
	var cfg config.Config
	cfg.Src.P4Port = src.Port()
	cfg.Src.P4User = src.User()
	cfg.Src.P4Client = src.Client()
	cfg.Dst.P4Port = dst.Port()
	cfg.Dst.P4User = dst.User()
	cfg.Dst.ClientName = dst.Client()
	cfg.Dst.ClientRoot = dst.Root()
	cfg.Dst.ClientStream = dst.StreamPath()

	return cfg.WriteToFile(file)
}

func runTwoServers(log frog.Logger, configSuffix int, src, dst Server, cl int64, expectedRuns int) error {
	log = frog.WithFields(log, frog.String("stage", "test"), frog.String("src", src.Port()), frog.String("dst", dst.Port()))
	l := frog.AddAnchor(log)
	defer frog.RemoveAnchor(l)

	shClose, sh := createBshWithTransientLogger(l)
	defer shClose()
	sh.SetVerbose(true)

	// on Windows you need a .exe at the end
	p4harmonize := sh.ExeName("p4harmonize")

	for i := 0; i < expectedRuns; i++ {

		// dst's client must not already exist
		if err := p4.New(sh, dst.Port(), dst.User(), "").DeleteClient(dst.Client()); err != nil {
			return fmt.Errorf("error deleting client %s from %s: %w", dst.Client(), dst.Port(), err)
		}
		// dst's root folder must be empty
		if err := os.RemoveAll(dst.Root()); err != nil {
			return fmt.Errorf("error deleting %s: %w", dst.Root(), err)
		}

		// write p4harmonize config and run
		cfgName := fmt.Sprintf("longtest_%d.toml", configSuffix)
		if err := writeConfig(cfgName, src, dst); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := sh.Cmdf(`./%s -config %s`, p4harmonize, cfgName).RunErr(); err != nil {
			return fmt.Errorf("p4harmonize with config %s: %w", cfgName, err)
		}

		p4src := p4.New(sh, src.Port(), src.User(), src.Client())
		p4dst := p4.New(sh, dst.Port(), dst.User(), dst.Client())

		// submit p4harmonize's changes
		if err := p4dst.SubmitChangelist(cl); err != nil {
			return fmt.Errorf("submit cl %d: %w", cl, err)
		}
		cl += 1

		srcFiles, dstFiles, err := buildDepotFilesLists(p4src, p4dst)
		if err != nil {
			return err
		}

		if expectedRuns == i+1 && srcFiles != dstFiles {
			log.Error("TEST FAILED: Source and Destination depots are not in sync")
			log.Error(" --SOURCE--")
			log.Error(srcFiles)
			log.Error(" --DESTINATION--")
			log.Error(dstFiles)
			return fmt.Errorf("expected src and dst files to match, but they did not")
		}

		if expectedRuns != i+1 && srcFiles == dstFiles {
			log.Error(fmt.Sprintf("TEST FAILED: Source and Destination fell into sync after %d run(s), but expected %d runs", i+1, expectedRuns))
			return fmt.Errorf("expected src and dst files to be different, but they matched")
		}
	}

	log.Info("Tests passed")

	return nil
}

func buildDepotFilesLists(p4src, p4dst *p4.P4) (srcFiles, dstFiles string, err error) {
	// helper to grab a list of files from one server
	getFilesAsString := func(pf *p4.P4) (string, error) {
		files, err := pf.ListDepotFiles()
		if err != nil {
			return "", err
		}
		sort.Sort(p4.DepotFileCaseInsensitive(files))
		var sb strings.Builder
		sb.Grow(32 * 1024)
		for _, v := range files {
			sb.WriteString(`   "`)
			sb.WriteString(v.Path)
			sb.WriteString(`" #`)
			sb.WriteString(v.Type)
			sb.WriteString(`" `)
			sb.WriteString(v.Digest)
			sb.WriteString("\n")
		}
		return sb.String(), nil
	}

	type result struct {
		pf   *p4.P4
		list string
		err  error
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var srcList, dstList string
	var srcErr, dstErr error

	go func() {
		defer wg.Done()
		srcList, srcErr = getFilesAsString(p4src)
	}()
	go func() {
		defer wg.Done()
		dstList, dstErr = getFilesAsString(p4dst)
	}()

	wg.Wait()

	if srcErr != nil && dstErr != nil {
		return "", "", fmt.Errorf("Errors listing depot files from both %s and %s: %v; %v", p4src.Port, p4dst.Port, srcErr, dstErr)
	}
	if srcErr != nil {
		return "", "", fmt.Errorf("Error listing depot files from %s: %w", p4src.Port, srcErr)
	}
	if dstErr != nil {
		return "", "", fmt.Errorf("Error listing depot files from %s: %w", p4dst.Port, dstErr)
	}

	return srcList, dstList, nil
}
