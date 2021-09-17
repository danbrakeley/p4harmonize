# p4harmonize

## TODO:

- for each file in the source file list (TODO: spread this work across multiple workers?)
  - if file is in destination
    - if destination casing doesn't match source
      - mark for rename (?)
    - else
      - mark for edit
  - copy file from source root to destination client root (which we just created, so we know it starts empty)
  - if file wasn't in destination CL, mark for add (use source casing)
  - ensure filetype is set correctly
- for each file in destination but NOT in source (add source files to map in previous loop?)
  - mark for delete

## Setup destination dev environment

Note: `p4harmonize` assumes you have bash locally (on Windows, just install Git for Windows, which includes git bash).

- Install Go 1.16 or newer.
- In bash (Git Bash on Windows), run `scripts/setup.sh`.
  - It may halt and ask you to install something. Once you've done that, run it again.
  - Repeat until it says "All dependancies are installed."

## Build/run

`p4harmonzie` uses [Mage](https://magefile.org), which is a build tool that runs Go code. The setup script from the previous step installs mage (in your `$GOPATH/bin` folder), so you should be able run `mage` to see what targets are available:

```text
$ mage
Targets:
  build    tests and builds the app (output goes to "local" folder)
  run      tests, builds, and runs the app
```

If you see `No .go files marked with the mage build tag in this directory.`, make sure you are in the root folder when running mage (mage does not currently look to parent folders for mage files).

The `build` and `run` targets will put the `p4harmonzie.exe` file in a subfolder named `local`, which is ignored by git. If you create a `config.toml` file in this folder, it will also be ignored by git.
