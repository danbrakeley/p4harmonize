# p4harmonize

## TODO:

- get epic client's root folder
- ensure epic root folder exists
- for each file in the epic file list (TODO: spread this work across multiple workers?)
  - if file is in local
    - if local casing doesn't match epic
      - mark for rename (?)
    - else
      - mark for edit
  - copy file from epic root to local client root (which we just created, so we know it starts empty)
  - if file wasn't in local CL, mark for add (use epic casing)
  - ensure filetype is set correctly
- for each file in local but NOT in epic (add epic files to map in previous loop?)
  - mark for delete
- clean up naming - don't use epic/local, instead use source/target? src/dst? read/write? something like that

## Setup local dev environment

Note: `p4harmonize` assumes you have bash locally (on Windows, just install Git for Windows, which includes git bash).

- Install Go 1.16 or newer.
- In bash (Git Bash on Windows), run `scripts/setup.sh`.
  - It may halt and ask you to install something. Once you've done that, run it again.
  - Repeat until it says "All dependancies are installed."

## Build/run

Once all dependancies are installed, open a fresh bash shell and cd into the root folder for p4harmonize (going into subfolders will cause `mage` to fail with `No .go files marked with the mage build tag in this directory.`).

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

$ mage build p4harmonize
Building p4harmonize.exe...
```

Each buildable app has its own subfolder in the `cmd` folder, so you can just `ls cmd` to list all the buildable targets:

```text
$ ls cmd
p4harmonize/  functests/
```

In the above case, you have two buildable targets: `mage build p4harmonize` and `mage build functests`.

The build command puts the executable in the `local` folder, and the run command cds into local, then runs the exe.
