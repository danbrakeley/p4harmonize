// +build mage

package main

import (
	"github.com/danbrakeley/bsh"
	"github.com/magefile/mage/mg"
)

var sh = &bsh.Bsh{}

// Builds named cmd (output goes to "local" folder).
func Build(cmd string) {
	target := sh.ExeName(cmd)

	sh.Echo("Running unit tests...")
	sh.Cmd("go test ./...").Run()

	sh.Echof("Building %s...", target)
	sh.MkdirAll("local/")
	sh.Cmdf("go build -o local/%s ./cmd/%s", target, cmd).Run()
}

// Removes all artifacts from previous builds.
// At the moment, this is accomplished by deleting the "local" folder.
func Clean() {
	sh.Echo("Deleting local...")
	sh.RemoveAll("local")
}

// Builds and runs named cmd.
func Run(cmd string) {
	mg.Deps(mg.F(Build, cmd))

	target := sh.ExeName(cmd)

	sh.Chdir("local")
	sh.Echo("Running...")
	sh.Cmdf("%s", target).Env(
		"VERBOSE=true",
	).Run()
}
