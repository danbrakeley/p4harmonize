# p4harmonize

## Overview

`p4harmonize` is a tool for getting the head revision of a stream on one perforce server to mirror the head revision from some other perforce server. It can reconcile files, fix differences in file name/path capitalization, fix the file type, and fix improperly checked in AppleDouble files created by the "apple" file type.

`p4harmonize` was built with Unreal Engine releases in mind, where the Epic licensee perforce server is used as the source, and a dedicated stream on a project's perforce server is used as the destination. It is intended to be used with a setup similar to [the one recommended by Epic](https://docs.unrealengine.com/4.26/en-US/ProgrammingAndScripting/ProgrammingWithCPP/DownloadingSourceCode/UpdatingSourceCode/#integrating,merging,andbranching). At Proletariat, we have different names, but the purposes are the same:

name | description
--- | ---
`//proj/engine_epic` | exact copy of a specific UE version from Epic's perforce server (p4harmonize targets this stream, no project-specific changes should ever end up here)
`//proj/engine_merge` | dedicated space for a human to merge main with new engine drops; ideally QA will sign off on a build from this branch before the merged engine changes are brought to main
`//proj/main` | our mainline; lots of people work here, so a broken build can be very costly

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

`p4harmonize` connects to each server, requests file lists from each, and determines what work needs to be done. If everything is already in sync, then it reports that back to the user and ends. If there is work to be done, then it creates a changelist and begins adding its fixes to it.

While it runs, it outputs status updates and every individual `p4` command it is running so you can follow along.

When it is done, there will be a changelist that must be submitted by hand. This gives you an opportunity to sanity check the changes before they are committed.

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
- Open a bash prompt in the created `p4harmonize` folder and run  `scripts/setup.sh`
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

- If your shell can't find `mage`, then you can try
  - re-run `scripts/setup.sh` (or `scripts/reinstall-mage.sh`), which creates a `mage` executable in your `$GOPATH/bin`
  - ensure you have a `$GOPATH/bin` folder, and that it has been added to your `$PATH` environment variable
- Mage targets are not case sensitive, so `mage longTest` and `mage longtest` will run the same target.
- If you see `No .go files marked with the mage build tag in this directory`, make sure you there is a `magefile.go` in the current folder (Mage does not look to parent folders for magefiles).

## Contributing

Before opening a pull request, please run `mage longtest`, which will run all tests, including functional tests that spin up two Perforce servers in Docker, populate each with some test files, then runs p4harmonize against them, and validates the results.

Once you have a passing longtest, feel free to open a PR. Please include a quick description of the problem your PR solves in the PR description.

### Debugging longtest

Unfortunately the specifics of `longtest` are a bit gross at the moment, and while I hope to one day have time to go in and clean things up, for now it is an all or nothing pass/fail, and so debugging often requires going in and temporarily altering it to focus on the specific behavior that is failing. See the `func LongTest()` in `magefile.go`.

## Special Thanks!

Thanks to [Bohdon Sayre](https://github.com/bohdon) and [Jørgen P. Tjernø](https://github.com/jorgenpt) for contributing time and code to help me fix my bugs and improve performance!

## TODO:

- clean up output to be more readable
- make longtest less gross to work with/debug
- test on other platforms (only tested on Windows so far)
