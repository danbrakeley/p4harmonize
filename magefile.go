//go:build mage

package main

import (
	"bytes"
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
	mg.SerialDeps(Build, TestDown, TestUp)

	sh.Cmdf("go run ./cmd/longtest/main.go").Run()

	sh.Echo("***")
	sh.Echo("*** Integration Test Passed!")
	sh.Echo("***")

	TestDown()
}

// TestUp brings up fresh perforce servers via Docker, each with a super user named "super" (no password).
func TestUp() {
	sh.InDir("test", func() {
		sh.Echo("Bringing up test perforce servers (see test/docker-compose.yaml for details)...")
		sh.Cmdf("docker compose up --detach --force-recreate --build").Run()
	})
}

// TestDown brings down and removes the docker containers started by TestUp.
func TestDown() {
	sh.InDir("test", func() {
		sh.Echo("Stopping and removing test perforce servers...")
		sh.Cmdf("docker compose stop -t 1").Run()
		sh.Cmdf("docker compose rm -f").Run()
	})
}
