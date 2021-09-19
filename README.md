# p4harmonize

## TODO:

- for each file in the source file list (TODO: spread this work across multiple workers?)
  - if file is in destination
    - copy file from src to dst
    - if destination casing doesn't match source
      - mark for rename
    - else
      - mark for edit
  - if file wasn't in destination CL
    - copy file from src to dst
    - mark for add
  - ensure filetype is set correctly?
- for each file in destination but NOT in source
  - mark for delete

## Runtime requirements

`p4harmonize` requires the following commands to be in your path:

- `p4`
  - Windows/Mac/Linux: [Helix Command-Line Client](https://www.perforce.com/downloads/helix-command-line-client-p4)
  - APT/YUM: [helix-cli](https://www.perforce.com/perforce-packages)
- `bash`
  - Windows: [Git for Windows](https://gitforwindows.org) includes git bash.
  - Mac/Linux: you should be all set

## Development setup

Make sure you have the following installed and in your path:

- `go`
  - Windows/Mac/Linux: [golang.org/dl](https://golang.org/dl/)
- `docker`
  - Windows/Mac: [Docker Desktop](https://www.docker.com/products/docker-desktop)
  - Linux: [Docker Server](https://docs.docker.com/engine/install/#server)

Grab the code:

```text
git clone https://github.com/proletariatgames/p4harmonize.git
```

From the created folder, run the setup script for magefile support:

```text
./scripts/setup.sh
```

If the setup script complains about something, fix it, then re-run the script until it says "All dependancies are installed."

## Build/run

`p4harmonzie` uses [Mage](https://magefile.org), which is a build tool that runs Go code. The setup script from the previous step installs mage (in your `$GOPATH/bin` folder), so you should be able run `mage` with no args to see what targets are available:

```text
$ mage
Targets:
  build       tests and builds the app (output goes to "local" folder)
  run         unit tests, builds, and runs the app
  testDown    kills and deletes the test perforce servers.
  testUp      brings up a test environment with two perforce servers, on ports 1667 and 1668, to act as the source and destination perforce servers for testing p4harmonize.
```

If you see `No .go files marked with the mage build tag in this directory.`, make sure you are in the root folder when running `mage` (`mage` does not currently look to parent folders for magefiles).

The `build` and `run` targets will put the `p4harmonzie.exe` file in a subfolder named `local`, which is ignored by git. You should create your `config.toml` in this folder as well.

Note that mage targets are not case sensative, ie `testUp`, `testup`, and `TESTUP` will all do the same thing.
