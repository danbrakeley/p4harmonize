package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/danbrakeley/bsh"
	"github.com/danbrakeley/frog"
	"github.com/proletariatgames/p4harmonize/internal/config"
	"github.com/proletariatgames/p4harmonize/internal/p4"
)

var Servers = []Server{
	{Src, 1661, "super", "UE4", "Release-4.20", "./p4/1"},
	{Src, 1662, "super", "UE4", "Release-4.20", "./p4/2"},
	{Dst, 1663, "super", "test", "engine", "./p4/3"},
	{Dst, 1664, "super", "test", "engine", "./p4/4"},
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
	log := frog.New(frog.Auto, frog.POTime(false), frog.POLevel(false))
	defer log.Close()

	// start := time.Now()

	// SETUP

	func() {
		log.Info(fmt.Sprintf("Populating %d perforce servers with test data...", len(Servers)))

		var wg sync.WaitGroup
		wg.Add(len(Servers))

		for _, v := range Servers {
			l := frog.AddAnchor(log)
			defer frog.RemoveAnchor(l)
			go func(log frog.Logger, s Server) {
				defer wg.Done()
				close, sh := frogBsh(log)
				defer close()
				pf := p4.New(sh, s.Port(), s.User(), "")
				if s.IsSrc() {
					setupSrc(sh, pf, s)
				} else {
					setupDst(sh, pf, s)
				}
				log.Info(fmt.Sprintf("Server %d ready", s.PortInt()))
			}(l, v)
		}

		wg.Wait()
	}()

	sh := &bsh.Bsh{}
	sh.SetVerbose(true)

	// on Windows you need a .exe at the end
	p4harmonize := sh.ExeName("p4harmonize")

	testFailure := false

	// RUN P4HARMONIZE

	func(src, dst Server) {
		log.Info(fmt.Sprintf("Running p4harmonize from %d to %d...", src.PortInt(), dst.PortInt()))

		cfgName := fmt.Sprintf("longtest_%d_%d.toml", src.PortInt(), dst.PortInt())

		sh.InDir("local", func() {
			// dst's client must not already exist
			sh.Must(p4.New(sh, dst.Port(), dst.User(), "").DeleteClient(dst.Client()))
			// dst's root folder must be empty
			sh.RemoveAll(dst.Root())

			// write p4harmonize config and run
			sh.Must(WriteConfig(cfgName, src, dst))
			sh.Cmdf(`./%s -config %s`, p4harmonize, cfgName).Run()
		})

		p4src := p4.New(sh, src.Port(), src.User(), src.Client())
		p4dst := p4.New(sh, dst.Port(), dst.User(), dst.Client())

		// submit p4harmonize's changes
		sh.Must(p4dst.SubmitChangelist(3))

		log.Info(fmt.Sprintf("Verifying depot files from %d and %d match...", src.PortInt(), dst.PortInt()))
		srcFiles, dstFiles, err := BuildDepotFilesLists(p4src, p4dst)
		sh.Must(err)

		if srcFiles != dstFiles {
			sh.Warn("TEST FAILED: Source and Destination depots are not in sync")
			sh.Warn(" --SOURCE--")
			sh.Warn(srcFiles)
			sh.Warn(" --DESTINATION--")
			sh.Warn(dstFiles)
			testFailure = true
		} else {
			log.Info("All depot files casing and type match!")
		}

	}(Servers[0], Servers[2])

	if testFailure {
		return 1
	}

	func(src, dst Server) {
		log.Info(fmt.Sprintf("Running p4harmonize from %d to %d...", src.PortInt(), dst.PortInt()))

		cfgName := fmt.Sprintf("longtest_%d_%d.toml", src.PortInt(), dst.PortInt())

		sh.InDir("local", func() {
			// dst's client must not already exist
			sh.Must(p4.New(sh, dst.Port(), dst.User(), "").DeleteClient(dst.Client()))
			// dst's root folder must be empty
			sh.RemoveAll(dst.Root())

			// write p4harmonize config and run
			sh.Must(WriteConfig(cfgName, src, dst))
			sh.Cmdf(`./%s -config %s`, p4harmonize, cfgName).Run()
		})

		p4src := p4.New(sh, src.Port(), src.User(), src.Client())
		p4dst := p4.New(sh, dst.Port(), dst.User(), dst.Client())

		// submit p4harmonize's changes
		sh.Must(p4dst.SubmitChangelist(3))

		// second pass
		sh.InDir("local", func() {
			// dst's client must not already exist
			sh.Must(p4.New(sh, dst.Port(), dst.User(), "").DeleteClient(dst.Client()))
			// dst's root folder must be empty
			sh.RemoveAll(dst.Root())

			// re-run p4harmonize
			sh.Cmdf(`./%s -config %s`, p4harmonize, cfgName).Run()
		})

		// submit p4harmonize's additional changes
		sh.Must(p4dst.SubmitChangelist(4))

		log.Info(fmt.Sprintf("Verifying depot files from %d and %d match...", src.PortInt(), dst.PortInt()))
		srcFiles, dstFiles, err := BuildDepotFilesLists(p4src, p4dst)
		sh.Must(err)

		if srcFiles != dstFiles {
			sh.Warn("TEST FAILED: Source and Destination depots are not in sync")
			sh.Warn(" --SOURCE--")
			sh.Warn(srcFiles)
			sh.Warn(" --DESTINATION--")
			sh.Warn(dstFiles)
			testFailure = true
		} else {
			log.Info("All depot files casing and type match!")
		}

	}(Servers[1], Servers[3])

	if testFailure {
		return 1
	}

	return 0
}

// connect a frog logger to a bsh shell, piping all output from bsh into the logger
func frogBsh(log frog.Logger) (close func(), sh *bsh.Bsh) {
	reader, writer := io.Pipe()
	close = func() { writer.Close() }

	// Start a goroutine that reads lines off the reader and logs them
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			log.Transient(line)
		}

		err := scanner.Err()
		if err != nil {
			log.Error("error in scanner", frog.Err(err))
		}
	}()

	sh = &bsh.Bsh{
		Stdout:       writer,
		DisableColor: true,
	}

	return
}

type ServerType uint8

const (
	Src ServerType = iota
	Dst ServerType = iota
)

type Server struct {
	t      ServerType
	port   int
	user   string
	depot  string
	stream string
	root   string
}

func (s *Server) IsSrc() bool {
	return s.t == Src
}

func (s *Server) IsDst() bool {
	return s.t == Dst
}

func (s *Server) Port() string {
	return strconv.Itoa(s.port)
}

func (s *Server) PortInt() int {
	return s.port
}

func (s *Server) User() string {
	return s.user
}

// depot name, e.g. "UE4"
func (s *Server) Depot() string {
	return s.depot
}

// stream name, e.g. "Release-4.20"
func (s *Server) StreamName() string {
	return s.stream
}

// fulll stream path, e.g. "//UE4/Release-4.20"
func (s *Server) StreamPath() string {
	return fmt.Sprintf("//%s/%s", s.depot, s.stream)
}

func (s *Server) Root() string {
	return s.root
}

func (s *Server) Client() string {
	return fmt.Sprintf("%s-%s-%s", s.user, s.depot, s.stream)
}

func setupSrc(sh *bsh.Bsh, pf *p4.P4, src Server) {
	sh.Must(pf.CreateStreamDepot(src.Depot()))
	sh.Must(pf.CreateMainlineStream(src.Depot(), src.StreamName()))
	stream := fmt.Sprintf("//%s/%s", src.Depot(), src.StreamName())
	srcRoot := filepath.Join("local", src.Root())
	sh.Must(pf.CreateStreamClient(src.Client(), srcRoot, stream))

	pf.Client = src.Client()
	cl, err := pf.CreateEmptyChangelist("longtest")
	sh.Must(err)

	sh.Echof("Created CL %d", cl)
	sh.RemoveAll(srcRoot)
	sh.MkdirAll(filepath.Join(srcRoot, "Engine", "Linux"))
	sh.MkdirAll(filepath.Join(srcRoot, "Engine", "Extras"))

	root := func(name string) string {
		return filepath.Join(srcRoot, name)
	}
	MustAddFile(sh, pf, cl, root("generate.cmd"), "binary", "echo foo")
	MustAddFile(sh, pf, cl, root("Engine/build.cs"), "text", "// build stuff")
	MustAddFile(sh, pf, cl, root("Engine/chair.uasset"), "binary+l", "I'm a chair!")
	MustAddFile(sh, pf, cl, root("Engine/door.uasset"), "binary+l", "I'm a door!")
	MustAddFile(sh, pf, cl, root("Engine/Linux/important.h"), "text", "#include <frank.h>")
	MustAddFile(sh, pf, cl, root("Engine/Linux/boring.h"), "text", "#include <greg.h>")
	MustAddFile(sh, pf, cl, root("Engine/Icon20@2x.png"), "binary", "¯\\_(ツ)_/¯")
	MustAddFile(sh, pf, cl, root("Engine/Icon30@2x.png"), "binary", "¯\\_(ツ)_/¯")
	MustAddFile(sh, pf, cl, root("Engine/Icon40@2x.png"), "binary", "¯\\_(ツ)_/¯")
	MustAddAppleFile(sh, pf, cl, root("Engine/Extras/Apple File.template"), "resource fork", "this is just the data fork")
	MustAddAppleFile(sh, pf, cl, root("Engine/Extras/Apple File Src.template"), "source fork", "this is just the data fork")
	MustAddAppleFile(sh, pf, cl, root("Engine/Extras/Borked.template"), "resource fork", "this is just the data fork")

	sh.Must(pf.SubmitChangelist(cl))
}

func setupDst(sh *bsh.Bsh, pf *p4.P4, dst Server) {
	sh.Must(pf.CreateStreamDepot(dst.Depot()))
	sh.Must(pf.CreateMainlineStream(dst.Depot(), dst.StreamName()))
	stream := fmt.Sprintf("//%s/%s", dst.Depot(), dst.StreamName())
	dstRoot := filepath.Join("local", dst.Root())
	sh.Must(pf.CreateStreamClient(dst.Client(), dstRoot, stream))

	pf.Client = dst.Client()
	cl, err := pf.CreateEmptyChangelist("longtest")
	sh.Must(err)

	sh.Echof("Created CL %d", cl)
	sh.RemoveAll(dstRoot)
	sh.MkdirAll(filepath.Join(dstRoot, "Engine", "linux")) // note lower case "l" on "linux"
	sh.MkdirAll(filepath.Join(dstRoot, "Engine", "Extras"))

	root := func(name string) string {
		return filepath.Join(dstRoot, name)
	}
	MustAddFile(sh, pf, cl, root("generate.cmd"), "text", "echo foo")
	MustAddFile(sh, pf, cl, root("deprecated.txt"), "utf8", "this file will be deleted very soon")
	MustAddFile(sh, pf, cl, root("Engine/build.cs"), "text", "// build stuff")
	MustAddFile(sh, pf, cl, root("Engine/chair.uasset"), "binary", "I'm a chair!")
	MustAddFile(sh, pf, cl, root("Engine/rug.uasset"), "binary", "I'm a rug!")
	MustAddFile(sh, pf, cl, root("Engine/linux/important.h"), "utf8", "#include <frank.h>")
	MustAddFile(sh, pf, cl, root("Engine/linux/boring.h"), "text", "#include <greg.h>")
	MustAddFile(sh, pf, cl, root("Engine/Icon30@2x.png"), "binary", "¯\\_(ツ)_/¯")
	MustAddFile(sh, pf, cl, root("Engine/Icon40@2x.png"), "binary", "image not found")
	MustAddAppleFile(sh, pf, cl, root("Engine/Extras/Apple File.template"), "i'm the resource fork", "this is just the data fork")
	MustAddAppleFile(sh, pf, cl, root("Engine/Extras/Apple File Dst.template"), "destination fork", "this is just the data fork")
	MustAddFile(sh, pf, cl, root("Engine/Extras/Borked.template"), "binary", "this is just the data fork")
	MustAddFile(sh, pf, cl, root("Engine/Extras/%Borked.template"), "binary", "this should never have been checked in")

	sh.Must(pf.SubmitChangelist(cl))
}

func MustAddFile(sh *bsh.Bsh, server *p4.P4, cl int64, filename, p4type, contents string) {
	sh.Must(AddFile(server, cl, filename, p4type, contents))
}

func AddFile(server *p4.P4, cl int64, filename, p4type, contents string) error {
	err := ioutil.WriteFile(filename, []byte(contents), 0666)
	if err != nil {
		return fmt.Errorf("error writing to %s: %w", filename, err)
	}
	return server.Add([]string{filename}, p4.Type(p4type), p4.Changelist(cl), p4.DoNotIgnore)
}

var doubleResourceHeader = [34]byte{
	0x00, 0x05, 0x16, 0x07, 0x00, 0x02, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00,
	0x00, 0x26,
}

func MustAddAppleFile(sh *bsh.Bsh, server *p4.P4, cl int64, filename, resource, data string) {
	sh.Must(AddAppleFile(server, cl, filename, resource, data))
}

func AddAppleFile(server *p4.P4, cl int64, filename, resource, data string) error {
	// An AppleDouble formatted file is actually made up of two files: one file containing
	// the resource fork, and another containing the data fork.

	// first build the contents of the resource fork file
	b := make([]byte, 0, len(doubleResourceHeader)+4+len(resource))
	b = append(b, doubleResourceHeader[:]...)
	b = binary.BigEndian.AppendUint32(b, uint32(len(resource)))
	b = append(b, []byte(resource)...)

	// then write the resource fork
	path := filepath.Join(filepath.Dir(filename), "%"+filepath.Base(filename))
	if err := ioutil.WriteFile(path, b, 0666); err != nil {
		return fmt.Errorf("error writing Apple Double resource fork to %s: %w", path, err)
	}

	// second write the data to the data fork file
	if err := ioutil.WriteFile(filename, []byte(data), 0666); err != nil {
		return fmt.Errorf("error writing Apple Double data fork to %s: %w", filename, err)
	}

	// AppleDouble files are added by a single call to p4 add (file type must be "apple")
	return server.Add([]string{filename}, p4.Type("apple"), p4.Changelist(cl), p4.DoNotIgnore)
}

func WriteConfig(file string, src Server, dst Server) error {
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

func BuildDepotFilesLists(p4src, p4dst *p4.P4) (srcFiles, dstFiles string, err error) {
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
