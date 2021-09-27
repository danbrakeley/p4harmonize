# p4harmonize

## Overview

`p4harmonize` is a tool for getting the head revision of a stream on one perforce server to mirror the head revision from some other perforce server. It can reconcile files, fix differences in file name/path capitalization, fix the file type, and fix improperly checked in AppleDouble files created by the "apple" file type.

`p4harmonize` was built with Unreal Engine releases in mind, where the Epic licensee perforce server is used as the source, and a dedicated stream on a project's perforce server is used as the destination. It is intended to be used with a setup similar to [the one recommended by Epic](https://docs.unrealengine.com/4.26/en-US/ProgrammingAndScripting/ProgrammingWithCPP/DownloadingSourceCode/UpdatingSourceCode/#option3:usingperforce). At Proletariat, we have different names, but the purposes are the same:

name | description
--- | ---
`//proj/engine_epic` | exact copy of a specific UE version from Epic's perforce server (p4harmonize targets this stream, no project-specific changes should ever end up here)
`//proj/engine_merge` | dedicated space for merging main with the new engine; QA needs to sign off before merging from here back into main
`//proj/main` | our mainline; lots of people work here, so a broken build can be very costly

## Install

There's no pre-packaged binaries at this point, so you'll need to build one yourself. To do that, you'll need [git](https://git-scm.com/downloads) and a [recent version of Go](https://golang.org/dl/), and then you can just use the [go install](https://golang.org/ref/mod#go-install) command to download, build, and put the result in your path:

```text
go install github.com/proletariatgames/p4harmonize
```

## Usage

`p4harmonize` pulls configuration from a `config.toml` file, which it looks for in the current folder. You can change where it looks by passing `-config <filename>`.

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

While it runs, it outputs status updates and every individual `p4` command it is running so you can follow along. Note that for an Unreal Engine upgrade, this process can easily take hours to complete.

When it is done, you still need to go in and submit the changelist it created yourself. This gives you an opportunity to sanity check the work before it gets added to your Perforce depot. This also allows you to keep the destination locked until the moment you are ready to submit the changes.

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

To write code and create builds, you just need the deps listed above (git, Go, p4, and bash). If you want to run the functional tests, where p4harmonize is run against two test Perforce servers, you will also need Docker. On Windows and Mac, you'll want [Docker Desktop](https://www.docker.com/products/docker-desktop), and on Linux you'll want [Docker Server](https://docs.docker.com/engine/install/#server).

Once you have all that installed, you can clone down the source code with git:

```text
git clone https://github.com/proletariatgames/p4harmonize.git
```

`p4harmonize` uses [Mage](https://magefile.org) to automate development tasks like building and running tests. You can run `./scripts/setup.sh` to install it.

## Build/run

There's a `magefile.go` in the root folder that automates building/testing, and running `mage` without any arguments will print the names of the targets it finds in that file:

```text
$ mage
Targets:
  build       tests and builds the app (output goes to "local" folder)
  longTest    Runs integration tests (spins up perforce servers via docker, then brings them down at the end).
  run         unit tests, builds, and runs the app
  testDown    kills and deletes the test perforce servers.
  testPrep    gets everything ready for a run against test servers, and can be used to reset test servers after a test run.
  testUp      brings up a test environment with two perforce servers, on ports 1667 and 1668, to act as the source and destination perforce servers for testing p4harmonize.
```

Note that mage target names are not case sensitive, ie `mage longTest` and `mage longtest` will all do the same thing.

If you see "No .go files marked with the mage build tag in this directory", make sure you there is a `magefile.go` in the current folder (`mage` does not look to parent folders for magefiles).

The `build` and `run` targets will run unit tests and build an executable in a folder named `local` (which is where it will look for a `config.toml` file).

To run a suite of integration tests against actual perforce servers, just run `mage longtest`. This will bring up two docker containers running perforce, and then run some scripts to populate each with different files, and then run p4harmonize against them and verify the results.

## TODO:

- UX pass: clean up output to be more readable
  - maybe log verbose to json file, while only printing >=Info on the command line?
  - add anchored lines to output?
- see if threading operations can speed up the total run time
  - upgrading UE4.26 to UE4.27 took just under 4h (which isn't terrible)
- test on other platforms (only tested on Windows so far)
