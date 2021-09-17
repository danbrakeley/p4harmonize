// +build mage

package main

import (
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
	sh.Cmdf("go build -o local/%s ./cmd/%s", target, cmd).Run()
}

// Run unit tests, builds, and runs the app
func Run() {
	mg.Deps(Build)

	target := sh.ExeName(cmd)

	sh.Chdir("local")
	sh.Echo("Running...")
	sh.Cmdf("%s", target).Env(
		"VERBOSE=true",
	).Run()
}

// TestUp brings up a test environment with two perforce servers, on ports 1667 and 1668, to act as the
// source and destination perforce servers for testing p4harmonize.
func TestUp() {
	sh.Chdir("test")
	sh.Echo("Running docker compose...")
	sh.Cmdf("docker compose up --detach --force-recreate --build").Run()
	sh.Echo("Running setup.sh...")
	sh.Cmdf("./setup.sh").Bash()
}

// TestDown kills and deletes the test perforce servers.
func TestDown() {
	sh.Chdir("test")
	sh.Echo("Stopping and removing containers...")
	sh.Cmdf("docker compose stop -t 1").Run()
	sh.Cmdf("docker compose rm -f").Run()
}
