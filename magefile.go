//go:build mage

package main

import (
	"bytes"
	"os"
	"strings"
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
	mg.Deps(Build)

	target := sh.ExeName(p4harmonize)

	sh.InDir("local", func() {
		sh.Echo("Running...")
		sh.Cmdf("%s", target).Run()
	})
}

// LongTest runs a fresh build of p4harmonize against test files in docker-hosted perforce servers.
func LongTest() {
	start := time.Now()
	lap := start
	defer func() {
		sh.Echof("total runing time: %v", time.Now().Sub(start))
	}()

	mg.SerialDeps(Build)
	sh.Echof("p4harmonize build time: %v", time.Now().Sub(lap))
	lap = time.Now()

	testUpCaptureOutput("local/longtest_docker.log")
	sh.Echof("docker time: %v", time.Now().Sub(lap))
	lap = time.Now()

	sh.Echo("Building longtest...")
	target := sh.ExeName("longtest")
	sh.Cmdf(`go build -o local/%s ./cmd/longtest`, target).Run()

	sh.Echof("longtest build time: %v", time.Now().Sub(lap))
	lap = time.Now()

	sh.InDir("local", func() {
		sh.Echo("Running longtest...")
		sh.Cmdf(`./%s`, target).Run()
	})

	sh.Warn("***")
	sh.Warn("*** Longtest succeeded! All tests passed!")
	sh.Warn("***")
	sh.Echof("longtest run time: %v", time.Now().Sub(lap))
	sh.Echo("Don't forget to run 'mage testdown' to bring down the servers")
}

// TestUp brings up fresh perforce servers via Docker, each with a super user named "super" (no password).
func TestUp() {
	sh.Echo("Bringing up test perforce servers (see test/docker-compose.yaml for details)...")
	testUp(sh)
}

// TestDown brings down and removes the docker containers started by TestUp.
func TestDown() {
	sh.InDir("test", func() {
		sh.Echo("Stopping and removing test perforce servers...")
		sh.Cmdf("docker compose stop -t 1").Run()
		sh.Cmdf("docker compose rm -f").Run()
	})
}

func testUp(sh *bsh.Bsh) {
	sh.InDir("test", func() {
		sh.Cmdf("docker compose up --detach --force-recreate --build").Run()
	})
}

func testUpCaptureOutput(logFile string) {
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
	testUp(shDockerLog)
}
