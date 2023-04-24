//go:build mage

package main

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/danbrakeley/bsh"
	"github.com/magefile/mage/mg"
)

var sh = &bsh.Bsh{}
var p4harmonize = "p4harmonize"

// Build tests and builds the app (output goes to "local" folder)
func Build() {
	target := sh.ExeName(p4harmonize)

	sh.Echo("Running unit tests...")
	sh.Cmd("go test ./...").Run()

	sh.Echof("Building %s...", target)
	sh.MkdirAll("local/")

	// grab git commit hash to use as version for local builds
	commit := "(dev)"
	var b bytes.Buffer
	n := sh.Cmd(`git log --pretty=format:'%h' -n 1`).Out(&b).RunExitStatus()
	if n == 0 {
		commit = strings.TrimSpace(b.String())
	}

	sh.Cmdf(
		`go build -ldflags '`+
			`-X "github.com/proletariatgames/p4harmonize/internal/buildvar.Version=%s" `+
			`-X "github.com/proletariatgames/p4harmonize/internal/buildvar.BuildTime=%s" `+
			`-X "github.com/proletariatgames/p4harmonize/internal/buildvar.ReleaseURL=https://github.com/proletariatgames/p4harmonize"`+
			`' -o local/%s ./cmd/%s`, commit, time.Now().Format(time.RFC3339), target, p4harmonize,
	).Run()
}

// Run runs unit tests, builds, and runs the app
func Run() {
	mg.SerialDeps(Build)

	target := sh.ExeName(p4harmonize)

	sh.Echo("Running...")
	sh.Cmdf("./%s", target).Dir("local").Run()
}

// LongTest runs a fresh build of p4harmonize against test files in docker-hosted perforce servers.
func LongTest() {
	start := time.Now()
	lap := start
	defer func() {
		sh.Echof("== total time: %v", time.Now().Sub(start))
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		mg.SerialDeps(Build)
	}()
	go func() {
		defer wg.Done()
		dockerPrep("local/longtest_docker.log")
	}()
	wg.Wait()

	sh.Echof("-- prep time: %v", time.Now().Sub(lap))
	lap = time.Now()

	sh.Echo("Running longtest...")
	sh.Cmdf("go run ../cmd/longtest/").Dir("local").Run()
	sh.Echof("-- longtest time: %v", time.Now().Sub(lap))
	lap = time.Now()

	sh.Warn("***")
	sh.Warn("*** Longtest succeeded! All tests passed!")
	sh.Warn("***")
	sh.Echo("Don't forget to run 'mage testdown' to bring down the servers")
}

// TestUp brings up fresh perforce servers via Docker, each with a super user named "super" (no password).
func TestUp() {
	sh.Echo("Bringing up test perforce servers (see test/docker-compose.yaml for details)...")
	testUp(sh)
}

// TestDown brings down and removes the docker containers started by TestUp.
func TestDown() {
	sh.Echo("Stopping and removing test perforce servers...")
	testDown(sh)
}

func testUp(sh *bsh.Bsh) {
	sh.Cmdf("docker compose up --detach --force-recreate --build").Dir("test").Run()
}

func testDown(sh *bsh.Bsh) {
	sh.Cmdf("docker compose stop -t 1").Dir("test").Run()
	sh.Cmdf("docker compose rm -f").Dir("test").Run()
}

func dockerPrep(logFile string) {
	sh.Echof("Bringing up test perforce servers (see '%s' for details)...", logFile)
	f, err := os.Create(logFile)
	sh.Must(err)
	defer func() {
		err := f.Close()
		if err != nil {
			sh.Warnf("Warning: error closing %s: %v", logFile, err)
		}
	}()

	shDockerLog := &bsh.Bsh{
		Stdout:       f,
		Stderr:       f,
		DisableColor: true,
	}
	shDockerLog.SetErrorHandler(func(err error) {
		sh.Warnf("Error bringing up perforce servers: %v", err)
		sh.Warnf("See '%s' for more info.", logFile)
		panic(err)
	})
	testDown(shDockerLog)
	testUp(shDockerLog)
}
