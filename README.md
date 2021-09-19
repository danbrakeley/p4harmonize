# p4harmonize

## TODO:

- recreate UE 4.26 to 4.27 upgrade as a final test case
- UX pass: output is a mess right now
  - maybe log verbose to json file, while only printing >=Info on the command line?
  - add anchored lines to output?
  - add progress during longer stretches? use 4.26-4.27 test to determine where this might be needed
- test on platforms (only tested on Windows so far)

## Install and Run

You'll need Go installed, at which point you can put a working exe in your path by doing:

```text
go install github.com/proletariatgames/p4harmonize
```

Then you run it in any folder, optionally specifying `-config <file-path>` to point to a config file. The default is `config.toml` in the current path. The config file looks like this:

```toml
## p4harmonize config

# source is the perforce server you are pulling from, ie Epic's licensee server.
[source]
p4port = "ssl:perforce.example.com:1667"
p4user = "user"
p4client = "user-UE4-Release-Latest-Minimal" # this needs to exist before running p4harmonize

# destination is the perforce server you want to update so that it matches the source
[destination]
p4port = "perforce.local:1666"
p4user = "localuser"
new_client_name = "localuser-harmonize"   # this will be created by p4harmonize
new_client_root = "d:/p4/local/harmonize" # this will be created by p4harmonize
new_client_stream = "//test/engine_epic"  # this needs to already exist
```

## Runtime requirements

`p4harmonize` requires the following commands to be in your path:

- `p4`
  - Windows/Mac/Linux: [Helix Command-Line Client](https://www.perforce.com/downloads/helix-command-line-client-p4)
  - APT/YUM: [helix-cli](https://www.perforce.com/perforce-packages)
- `bash`
  - Windows: [Git for Windows](https://gitforwindows.org) includes git bash.
  - Mac/Linux: you should be all set
  - the need for this can probably be removed in the future, it was just quicker to shell out to bash in some cases

## Development setup

Make sure you have the following installed and in your path:

- `go`
  - Windows/Mac/Linux: [golang.org/dl](https://golang.org/dl/)
- `docker` (needed for testing against throw-away perforce servers)
  - Windows/Mac: [Docker Desktop](https://www.docker.com/products/docker-desktop)
  - Linux: [Docker Server](https://docs.docker.com/engine/install/#server)

Grab the code:

```text
git clone https://github.com/proletariatgames/p4harmonize.git
```

From the created folder, run the setup script:

```text
./scripts/setup.sh
```

If the setup script complains about something missing, install that something, then re-run the script. Once the script says, "All dependancies are installed", you are all set.

## Build/run

`p4harmonzie` uses [Mage](https://magefile.org), which is a build tool that uses Go code. The setup script from the previous step installs mage, so you should be able run `mage` with no args to see what targets are available:

```text
$ mage
Targets:
  build       tests and builds the app (output goes to "local" folder)
  run         unit tests, builds, and runs the app
  test        Run unit tests, builds, and runs the app against the test servers
  testDown    kills and deletes the test perforce servers.
  testPrep    gets everything ready for a run against test servers, and can be used to reset test servers after a test run.
  testUp      brings up a test environment with two perforce servers, on ports 1667 and 1668, to act as the source and destination perforce servers for testing p4harmonize.
```

If you see `No .go files marked with the mage build tag in this directory.`, make sure you are in the root folder (`mage` does not look to parent folders for magefiles).

The `build` and `run` targets will put the `p4harmonzie.exe` file in a subfolder named `local`, which is ignored by git. You should create your `config.toml` in this folder as well, which is what will be used by `run`.

To run `p4harmonize` against test servers, run `mage testprep`, then `mage test`. To reset the perforce servers back to the starting state, run `mage testprep` again. When you are done, you can shut down the perforce servers by running `mage testdown`.

While the test perforce servers are up, you can connect with p4v by connecting to local ports `1667` and `1668` with user `super` (no password, so no login needed).

Note that mage targets are not case sensative, ie `testUp`, `testup`, and `TESTUP` will all do the same thing.
