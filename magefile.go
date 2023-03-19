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
var cmd = "p4harmonize"

// Build tests and builds the app (output goes to "local" folder)
func Build() {
	target := sh.ExeName(cmd)

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
			`' -o local/%s ./cmd/%s`, commit, time.Now().Format(time.RFC3339), target, cmd,
	).Run()
}

// Run runs unit tests, builds, and runs the app
func Run() {
	mg.Deps(Build)

	target := sh.ExeName(cmd)

	sh.InDir("local", func() {
		sh.Echo("Running...")
		sh.Cmdf("%s", target).Run()
	})
}

// LongTest runs a fresh build of p4harmonize against test files in docker-hosted perforce servers.
func LongTest() {
	mg.SerialDeps(Build, TestPrep)

	target := sh.ExeName(cmd)

	sh.InDir("local", func() {
		sh.Echof("Running %s against test servers...", target)
		sh.Cmdf("./%s -config ../test/config.toml", target).Run()
	})
	sh.InDir("test", func() {
		sh.Echo("Submitting p4harmonize's CL...")
		sh.Cmdf("./submit.sh").Bash()
		sh.Echo("Verifying depot files in both servers match...")
		sh.Cmdf("./verify.sh").Bash()
	})

	sh.Echo("***")
	sh.Echo("*** Integration Test Passed!")
	sh.Echo("***")
	TestDown()
}

// TestPrep runs testDown, then testUp, then executes `test/prep.sh` to fill the servers with test data.
func TestPrep() {
	mg.SerialDeps(TestDown, TestUp)
	sh.InDir("test", func() {
		sh.Echo("Running test/prep.sh...")

		// TODO: FIXME: This sleep is to reduce the chance of a race condition where prep.sh runs before
		// the perforce servers are actually accepting connections. I've only seen this issue on linux
		// (which was running in Windows via VMWare), and it was inconsistent.
		// Ideally there would be some check we could make here instead of just waiting and hoping.
		time.Sleep(time.Second)

		sh.Cmdf("./prep.sh").Bash()
	})
	sh.RemoveAll("./local/p4/dst")
}

// TestUp brings up two empty perforce servers via Docker, listening on ports 1667 and 1668, with
// a single super user named "super" (no password).
func TestUp() {
	sh.InDir("test", func() {
		sh.Echo("Bringing up test perforce servers on local ports 1667 and 1668...")
		sh.Cmdf("docker compose up --detach --force-recreate --build").Run()
	})
}

// TestDown brings down and removes the docker contains started by TestUp.
func TestDown() {
	sh.InDir("test", func() {
		sh.Echo("Stopping and removing test perforce servers...")
		sh.Cmdf("docker compose stop -t 1").Run()
		sh.Cmdf("docker compose rm -f").Run()
	})
}
