// +build mage

package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danbrakeley/bsh"
	"github.com/magefile/mage/mg"
)

var sh = &bsh.Bsh{}
var cmd = "p4harmonize"

// Build tests and builds the app (output goes to "local" folder)
func Build() {
	target := sh.ExeName(cmd)

	sh.Echo("Running unit tests...")
	sh.Cmd("go test ./...").Run()

	sh.Echof("Building %s...", target)
	sh.MkdirAll("local/")
	sh.Cmdf("go build -o local/%s ./cmd/%s", target, cmd).Run()
}

// Run runs unit tests, builds, and runs the app
func Run() {
	mg.Deps(Build)

	target := sh.ExeName(cmd)

	sh.InDir("local", func() {
		sh.Echo("Running...")
		sh.Cmdf("%s", target).Run()
	})
}

// LongTest runs a fresh build of p4harmonize against test files in docker-hosted perforce servers.
func LongTest() {
	mg.SerialDeps(Build, TestPrep)
	defer TestDown()

	target := sh.ExeName(cmd)

	sh.InDir("local", func() {
		sh.Echo("Running p4harmonize against test servers...")
		sh.Cmdf("%s -config ../test/config.toml", target).Run()
	})
	sh.InDir("test", func() {
		sh.Echo("Submitting CL and verifying depot files in both servers match...")
		sh.Cmdf("./verify.sh").Bash()
	})

	// The depot paths and types match, now let's check the file sizes and contents:
	sh.Echo("Comparing actual files (not just what perforce reports)...")
	sh.Must(compareFolderContents("./local/p4/src", "./local/p4/dst"))
	sh.Echo("All file sizes and contents match")
	sh.Echo("***")
	sh.Echo("*** Integration Test Passed!")
	sh.Echo("***")
}

// TestPrep runs testDown, then testUp, then executes `test/prop.sh` to fill the servers with test data.
func TestPrep() {
	mg.SerialDeps(TestDown, TestUp)
	sh.InDir("test", func() {
		sh.Echo("Running test/prep.sh...")
		sh.Cmdf("./prep.sh").Bash()
	})
	sh.RemoveAll("./local/p4/dst")
}

// TestUp brings up two empty perforce servers via Docker, listening on ports 1667 and 1668, with
// a single super user named "super" (no password).
func TestUp() {
	sh.InDir("test", func() {
		sh.Echo("Bringing up test perforce servers on local ports 1667 and 1668...")
		sh.Cmdf("docker compose up --detach --force-recreate --build").Run()
	})
}

// TestDown brings down and removes the docker contains started by TestUp.
func TestDown() {
	sh.InDir("test", func() {
		sh.Echo("Stopping and removing test perforce servers...")
		sh.Cmdf("docker compose stop -t 1").Run()
		sh.Cmdf("docker compose rm -f").Run()
	})
}

// helpers

type File struct {
	Name string
	Size int64
	Hash string
}

func (l File) IsSameAs(r File) bool {
	return strings.ToLower(l.Name) == strings.ToLower(r.Name) && l.Size == r.Size && l.Hash == r.Hash
}

type byNameLC []File

func (x byNameLC) Len() int           { return len(x) }
func (x byNameLC) Less(i, j int) bool { return strings.ToLower(x[i].Name) < strings.ToLower(x[j].Name) }
func (x byNameLC) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func compareFolderContents(src, dst string) error {
	srcFiles, err := getFolderContentsSorted(src)
	if err != nil {
		return err
	}
	dstFiles, err := getFolderContentsSorted(dst)
	if err != nil {
		return err
	}

	ok := len(srcFiles) == len(dstFiles)
	if ok {
		for i := range srcFiles {
			if !srcFiles[i].IsSameAs(dstFiles[i]) {
				ok = false
				break
			}
		}
	}

	if !ok {
		sh.Warnf("Mismatch between src (%d) and dst (%d):", len(srcFiles), len(dstFiles))
		sh.Warn("SOURCE:")
		for _, v := range srcFiles {
			sh.Warnf("   %v", v)
		}
		sh.Warn("DESTINATION:")
		for _, v := range dstFiles {
			sh.Warnf("   %v", v)
		}
		return fmt.Errorf("Test Failed")
	}
	return nil
}

func getFolderContentsSorted(root string) ([]File, error) {
	cleanRoot := filepath.Clean(root)
	var out []File
	err := filepath.Walk(cleanRoot, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		hash, err := hashFile(path)
		if err != nil {
			return fmt.Errorf("error hashing '%s': %w", path, err)
		}
		out = append(out, File{
			Name: strings.TrimPrefix(path, cleanRoot),
			Size: info.Size(),
			Hash: hash,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Sort(byNameLC(out))
	return out, err
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
