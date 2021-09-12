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

// Run tests, builds, and runs the app
func Run() {
	mg.Deps(Build)

	target := sh.ExeName(cmd)

	sh.Chdir("local")
	sh.Echo("Running...")
	sh.Cmdf("%s", target).Env(
		"VERBOSE=true",
	).Run()
}
