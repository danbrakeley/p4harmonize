# p4harmonize

## Overview

`p4harmonize` is a tool for mirroring a stream's head revision from one perforce server to another. This includes reconciling files, fixing differences in file name/path capitalization, fixing the file type, and fixing improperly checked in %AppleDouble files.

`p4harmonize` was built with Unreal Engine source in mind, where the Epic licensee perforce server is used as the source, and a dedicated stream on a project's perforce server is used as the destination. It is intended to be used with a setup similar to [the one recommended by Epic](https://docs.unrealengine.com/4.26/en-US/ProgrammingAndScripting/ProgrammingWithCPP/DownloadingSourceCode/UpdatingSourceCode/#integrating,merging,andbranching). At Proletariat, we have different names, but the purposes are the same:

name | description
--- | ---
`//proj/engine_epic` | exact copy of a specific UE version from Epic's perforce server (p4harmonize targets this stream, no project-specific changes should ever end up here)
`//proj/engine_merge` | dedicated space for a human to merge main with new engine drops; ideally QA will sign off on a build from this branch before the merged engine changes are brought to main
`//proj/main` | our mainline; lots of people work here, so a broken build can be very costly

## Case-sensitivity

As of v0.4.0, `p4harmonize` is doesn't support a destination Perforce server that is set to case-insensitive. If you'd like to help investigate and fix the issues around case-insesnsitive servers, then the best place to start is to run `mage longtest` and ensure it passes, then edit `test/docker-compose.yaml` to set the `p4dst` server's `CASE_INSENSITIVE` arg to `1`, and run `mage longtest` again, and now it should fail.

## Install

You can download the latest Windows executable from the [releases page](https://github.com/proletariatgames/p4harmonize/releases), or you can build it yourself.

To build your own, see the [Developement Setup](#development-setup) section below.

## Usage

`p4harmonize` requires a configuration TOML file, and by default will look for `config.toml` in the current directory. This can be overridden by passing `-config <file>`.

Here's an example `config.toml` file:

```toml
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

`p4harmonize` connects to each server, requests file lists from each, and determines what work needs to be done. If everything is already in sync, then it quickly reports the status and stops. If there is work to be done, then it creates a changelist and begins adding its fixes to it.

While it runs, it outputs status updates and every individual `p4` command it is running so you can follow along.

When it is done, there will be a changelist that must be submitted by hand, giving you a chance to sanity check the work.

## Runtime requirements

`p4harmonize` requires the following commands to be in your path:

- `p4`/`p4.exe`, aka the [Helix Command-Line Client](https://www.perforce.com/downloads/helix-command-line-client-p4)
- `bash`
  - On Mac/Linux, this comes standard
  - On Windows, I recommend git-bash, which is included in [Git for Windows](https://gitforwindows.org).

## Development setup

- Ensure you have all the [Runtime requirements](#runtime-requirements)
- Ensure you have a [recent version of Go (1.17+)](https://go.dev/dl/)
  - Ensure that $GOPATH/bin is added to your $PATH environment variable.
- To run all the tests, you'll want to have Docker installed
  - On Windows and Mac, you'll want [Docker Desktop](https://www.docker.com/products/docker-desktop)
  - On Linux you'll want [Docker Server](https://docs.docker.com/engine/install/#server).
- Clone the github repo:
  ```text
  git clone https://github.com/proletariatgames/p4harmonize.git
  ```
- Open a bash prompt in the `p4harmonize` folder that git created in the previous step and run `scripts/setup.sh`
  - This ensures Go is installed and ready
  - This builds [Mage](https://magefile.org) into your $GOPATH/bin folder

## Build/run

`p4harmonize` uses [Mage](https://magefile.org) to automate building and testing tasks. If you've never used Mage before, running `mage` with no arguments will display all the build targets found in the `magefile.go` file in the current folder. It should look something like this:

```text
$ mage
Targets:
  build       tests and builds the app (output goes to "local" folder)
  longTest    runs a fresh build of p4harmonize against test files in docker-hosted perforce servers.
  run         runs unit tests, builds, and runs the app
  testDown    brings down and removes the docker contains started by TestUp.
  testPrep    runs testDown, then testUp, then executes `test/prop.sh` to fill the servers with test data.
  testUp      brings up two empty perforce servers via Docker, listening on ports 1667 and 1668, with a single super user named "super" (no password).
```

### Mage usage notes

- If your shell can't find `mage`
  - make sure your `$GOPATH/bin` folder is in your `$PATH`
  - make sure there's a `mage` executable in your `$GOPATH/bin` folder
  - if `mage` is missing, then re-run `scripts/setup.sh`
- Mage targets are not case sensitive, so `mage longTest` and `mage longtest` will run the same target.

## Contributing

Before opening a pull request, please run `mage longtest`, which spins up two Perforce servers via Docker, places test files in them, then runs p4harmonize and validates the results.

Once `longtest` is passing, feel free to open a PR. Please include a quick description of the problem your PR solves in the PR description. If your PR includes performance improvements, please include benchmark numbers and an explanation of how to reproduce those numbers.

### Debugging longtest

Unfortunately `longtest` is a bit gross at the moment, and while I hope to one day have time to go in and break it out into individual test cases, for now it is an all or nothing pass/fail. Debugging failures often requires temporarily altering the bash scripts and/or connecting perforce clients directly to the docker servers to find and fix the specific behavior that is failing. A good starting point is to read through `func LongTest()` in `magefile.go`.

## Special Thanks!

Thanks to [Bohdon Sayre](https://github.com/bohdon) and [Jørgen P. Tjernø](https://github.com/jorgenpt) for contributing time and code to help me fix my bugs and dramatically improve performance!

## TODO:

- fix issues with case-insensitive destination servers
- clean up output to be more readable
- make longtest less gross to work with/debug
- run longtest via github actions?
- test on a Mac (maybe with github actions?)
