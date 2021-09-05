// +build mage

package main

import (
	"github.com/danbrakeley/bs"
	"github.com/magefile/mage/mg"
)

// Builds named cmd (output goes to "local" folder).
func Build(cmd string) {
	target := bs.ExeName(cmd)

	bs.Echo("Running unit tests...")
	bs.Cmd("go test ./...").Run()

	bs.Echof("Building %s...", target)
	bs.MkdirAll("local/")
	bs.Cmdf("go build -o local/%s ./cmd/%s", target, cmd).Run()
}

// Removes all artifacts from previous builds.
// At the moment, this is accomplished by deleting the "local" folder.
func Clean() {
	bs.Echo("Deleting local...")
	bs.RemoveAll("local")
}

// Builds and runs named cmd.
func Run(cmd string) {
	mg.Deps(mg.F(Build, cmd))

	target := bs.ExeName(cmd)

	bs.Chdir("local")
	bs.Echo("Running...")
	bs.Cmdf("%s", target).Env(
		"VERBOSE=true",
	).Run()
}
