# p4drop

## Setup local dev environment

- In git-bash, run `scripts/setup.sh`.
  - It may halt and ask you to install something. Once you've done that, run it again.
  - Repeat until it says "All dependancies are installed."

## Build/run

Once all dependancies are installed, open a fresh bash shell and cd into the root folder for p4drop (going into subfolders will cause `mage` to fail with `No .go files marked with the mage build tag in this directory.`).

Running `mage` with no arguments will list all available targets:

```text
$ mage
Targets:
  build    Builds named cmd (output goes to "local" folder).
  clean    Removes all artifacts from previous builds.
  run      Builds and runs named cmd.
```

You can add/remove/modify targets by adding/removing/modifying the functions in `magefile.go`.

Some targets take arguments, for example:

```text
$ mage build
not enough arguments for target "Build", expected 1, got 0

$ mage build p4drop
Building p4drop.exe...
```

Each buildable app has its own subfolder in the `cmd` folder, so you can just `ls cmd` to list all the buildable targets:

```text
$ ls cmd
p4drop/  functests/
```

In the above case, you have two buildable targets: `mage build p4drop` and `mage build functests`.

The build command puts the executable in the `local` folder, and the run command cds into local, then runs the exe.
