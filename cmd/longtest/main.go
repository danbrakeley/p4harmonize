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

// var sh = &bsh.Bsh{}

const (
	USER = "super"

	SRC_DEPOT  = "UE4"
	SRC_STREAM = "Release-4.20"
	SRC_ROOT   = "local/p4/src"

	DST_DEPOT  = "test"
	DST_STREAM = "engine"
	DST_ROOT   = "local/p4/dst"

	DST_INS_ROOT = "local/p4/dst_ins"
)

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
	log.Info("----- longtest begin!")

	func() {
		log.Info("Populating perforce servers at 1667, 1668, and 1669 with test data...")

		var logs [3]frog.Logger
		logs[0] = frog.AddAnchor(log)
		defer frog.RemoveAnchor(logs[0])
		logs[1] = frog.AddAnchor(log)
		defer frog.RemoveAnchor(logs[1])
		logs[2] = frog.AddAnchor(log)
		defer frog.RemoveAnchor(logs[2])

		var wg sync.WaitGroup
		wg.Add(3)

		go func(log frog.Logger) {
			defer wg.Done()
			close, sh := frogBsh(log)
			defer close()
			p4src := p4.New(sh, "1667", USER, "")
			setupSrc(sh, &p4src, SRC_ROOT)
			log.Info("Server 1667 ready")
		}(logs[0])

		go func(log frog.Logger) {
			defer wg.Done()
			close, sh := frogBsh(log)
			defer close()
			p4dst := p4.New(sh, "1668", USER, "")
			setupDst(sh, &p4dst, DST_ROOT)
			log.Info("Server 1668 ready")
		}(logs[1])

		go func(log frog.Logger) {
			defer wg.Done()
			close, sh := frogBsh(log)
			defer close()
			p4dst_ins := p4.New(sh, "1669", USER, "")
			setupDst(sh, &p4dst_ins, DST_INS_ROOT)
			log.Info("Server 1669 ready")
		}(logs[2])

		wg.Wait()
	}()

	sh := &bsh.Bsh{}
	testFailure := false

	func() {
		log.Info("Running p4harmonize from 1667 to 1668...")
		sh.SetVerbose(true)

		// on Windows you need a .exe at the end
		p4harmonize := sh.ExeName("p4harmonize")

		sh.InDir("local", func() {
			sh.RemoveAll("p4/dst")
			sh.Must(WriteConfig("longtest.toml", 1668, "./p4/dst"))
			sh.Cmdf(`./%s -config longtest.toml`, p4harmonize).Run()
		})

		p4src := p4.New(sh, "1667", USER, fmt.Sprintf("%s-%s-%s", USER, SRC_DEPOT, SRC_STREAM))
		p4dst := p4.New(sh, "1668", USER, fmt.Sprintf("%s-%s-%s-p4harmonize", USER, DST_DEPOT, DST_STREAM))

		// submit p4harmonize's changes
		sh.Must(p4dst.SubmitChangelist(3))

		// grab list of files from both servers
		getFilesAsString := func(pf *p4.P4) string {
			files, err := pf.ListDepotFiles()
			sh.Must(err)
			sort.Sort(p4.DepotFileCaseInsensitive(files))
			var sb strings.Builder
			sb.Grow(32 * 1024)
			for _, v := range files {
				sb.WriteString("   ")
				sb.WriteString(v.Path)
				sb.WriteString("   #")
				sb.WriteString(v.Type)
				sb.WriteString("\n")
			}
			return sb.String()
		}

		src := getFilesAsString(&p4src)
		dst := getFilesAsString(&p4dst)

		if src != dst {
			sh.Warn("TEST FAILED: Source and Destination depots are not in sync")
			sh.Warn(" --SOURCE--")
			sh.Warn(src)
			sh.Warn(" --DUSTINATION--")
			sh.Warn(dst)
			testFailure = true
		}

	}()

	if testFailure {
		return 1
	}

	func() {
		log.Info("Running p4harmonize from 1667 to 1669...")
		sh.SetVerbose(true)

		// on Windows you need a .exe at the end
		p4harmonize := sh.ExeName("p4harmonize")

		sh.InDir("local", func() {
			sh.RemoveAll("p4/dst_ins")
			sh.Must(WriteConfig("longtest.toml", 1669, "./p4/dst_ins"))
			sh.Cmdf(`./%s -config longtest.toml`, p4harmonize).Run()
		})

		p4src := p4.New(sh, "1667", USER, fmt.Sprintf("%s-%s-%s", USER, SRC_DEPOT, SRC_STREAM))
		p4dst := p4.New(sh, "1669", USER, fmt.Sprintf("%s-%s-%s-p4harmonize", USER, DST_DEPOT, DST_STREAM))

		// submit p4harmonize's changes
		sh.Must(p4dst.SubmitChangelist(3))

		{
			p4tmp := p4.New(sh, "1669", USER, "")
			sh.Must(p4tmp.DeleteClient(p4dst.Client))
		}

		// second pass
		sh.InDir("local", func() {
			sh.RemoveAll("p4/dst_ins")
			sh.Cmdf(`./%s -config longtest.toml`, p4harmonize).Run()
		})

		// submit p4harmonize's changes
		sh.Must(p4dst.SubmitChangelist(4))

		// grab list of files from both servers
		getFilesAsString := func(pf *p4.P4) string {
			files, err := pf.ListDepotFiles()
			sh.Must(err)
			sort.Sort(p4.DepotFileCaseInsensitive(files))
			var sb strings.Builder
			sb.Grow(32 * 1024)
			for _, v := range files {
				sb.WriteString("   ")
				sb.WriteString(v.Path)
				sb.WriteString("   #")
				sb.WriteString(v.Type)
				sb.WriteString("\n")
			}
			return sb.String()
		}

		src := getFilesAsString(&p4src)
		dst := getFilesAsString(&p4dst)

		if src != dst {
			sh.Warn("TEST FAILED: Source and Destination depots are not in sync")
			sh.Warn(" --SOURCE--")
			sh.Warn(src)
			sh.Warn(" --DUSTINATION--")
			sh.Warn(dst)
			testFailure = true
		}

	}()

	if testFailure {
		return 1
	}

	log.Info("----- success!")

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

func setupSrc(sh *bsh.Bsh, pf *p4.P4, srcRoot string) {
	sh.Must(pf.CreateStreamDepot(SRC_DEPOT))
	sh.Must(pf.CreateMainlineStream(SRC_DEPOT, SRC_STREAM))
	client := fmt.Sprintf("%s-%s-%s", USER, SRC_DEPOT, SRC_STREAM)
	stream := fmt.Sprintf("//%s/%s", SRC_DEPOT, SRC_STREAM)
	sh.Must(pf.CreateStreamClient(client, srcRoot, stream))

	pf.Client = client
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

func setupDst(sh *bsh.Bsh, pf *p4.P4, dstRoot string) {
	sh.Must(pf.CreateStreamDepot(DST_DEPOT))
	sh.Must(pf.CreateMainlineStream(DST_DEPOT, DST_STREAM))
	client := fmt.Sprintf("%s-%s-%s", USER, DST_DEPOT, DST_STREAM)
	stream := fmt.Sprintf("//%s/%s", DST_DEPOT, DST_STREAM)
	sh.Must(pf.CreateStreamClient(client, dstRoot, stream))

	pf.Client = client
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

func WriteConfig(file string, dstPort int, dstPath string) error {
	var cfg config.Config
	cfg.Src.P4Port = "1667"
	cfg.Src.P4User = "super"
	cfg.Src.P4Client = "super-UE4-Release-4.20"
	cfg.Dst.P4Port = strconv.Itoa(dstPort)
	cfg.Dst.P4User = "super"
	cfg.Dst.ClientName = "super-test-engine-p4harmonize"
	cfg.Dst.ClientRoot = dstPath
	cfg.Dst.ClientStream = "//test/engine"

	return cfg.WriteToFile(file)
}
