package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/danbrakeley/bsh"
	"github.com/danbrakeley/p4harmonize/internal/p4"
)

func setupCommon(pf *p4.P4, srv Server) (cl int64, err error) {
	if err := pf.CreateStreamDepot(srv.Depot()); err != nil {
		return -1, err
	}
	if err := pf.CreateMainlineStream(srv.Depot(), srv.StreamName()); err != nil {
		return -1, err
	}
	if err := pf.CreateStreamClient(srv.Client(), srv.Root(), srv.StreamPath()); err != nil {
		return -1, err
	}

	pf.Client = srv.Client()
	return pf.CreateEmptyChangelist("longtest")
}

func setupSrc(sh *bsh.Bsh, pf *p4.P4, src Server) error {
	cl, err := setupCommon(pf, src)
	if err != nil {
		return err
	}

	sh.Echof("Created CL %d", cl)
	if err := os.RemoveAll(src.Root()); err != nil {
		return err
	}

	for _, v := range []struct {
		Filename string
		Type     string
		Contents string
	}{
		{"generate.cmd", "binary", "echo foo"},
		{"Engine/build.cs", "text", "// build stuff"},
		{"Engine/chair.uasset", "binary+l", "I'm a chair!"},
		{"Engine/door.uasset", "binary+l", "I'm a door!"},
		{"Engine/Linux/important.h", "text", "#include <frank.h>"},
		{"Engine/Linux/boring.h", "text", "#include <greg.h>"},
		{"Engine/Icon20@2x.png", "binary", "¯\\_(ツ)_/¯"},
		{"Engine/Icon30@2x.png", "binary", "¯\\_(ツ)_/¯"},
		{"Engine/Icon40@2x.png", "binary", "¯\\_(ツ)_/¯"},
	} {
		if err := addFile(pf, cl, filepath.Join(src.Root(), v.Filename), v.Type, v.Contents); err != nil {
			return err
		}
	}

	for _, v := range []struct {
		Filename string
		Resource string
		Data     string
	}{
		{"Engine/Extras/Apple File.template", "resource fork", "this is just the data fork"},
		{"Engine/Extras/Apple File Src.template", "source fork", "this is just the data fork"},
		{"Engine/Extras/Borked.template", "resource fork", "this is just the data fork"},
	} {
		if err := addAppleFile(pf, cl, filepath.Join(src.Root(), v.Filename), v.Resource, v.Data); err != nil {
			return err
		}
	}

	if err := pf.SubmitChangelist(cl); err != nil {
		return err
	}

	return nil
}

func setupDst(sh *bsh.Bsh, pf *p4.P4, dst Server) error {
	cl, err := setupCommon(pf, dst)
	if err != nil {
		return err
	}

	sh.Echof("Created CL %d", cl)
	if err := os.RemoveAll(dst.Root()); err != nil {
		return err
	}

	for _, v := range []struct {
		Filename string
		Type     string
		Contents string
	}{
		{"generate.cmd", "text", "echo foo"},
		{"deprecated.txt", "utf8", "this file will be deleted very soon"},
		{"Engine/build.cs", "text", "// build stuff"},
		{"Engine/chair.uasset", "binary", "I'm a chair!"},
		{"Engine/rug.uasset", "binary", "I'm a rug!"},
		{"Engine/linux/important.h", "utf8", "#include <frank.h>"},
		{"Engine/linux/boring.h", "text", "#include <greg.h>"},
		{"Engine/Icon30@2x.png", "binary", "¯\\_(ツ)_/¯"},
		{"Engine/Icon40@2x.png", "binary", "image not found"},
		{"Engine/Extras/Borked.template", "binary", "this is just the data fork"},
		{"Engine/Extras/%Borked.template", "binary", "this should never have been checked in"},
	} {
		if err := addFile(pf, cl, filepath.Join(dst.Root(), v.Filename), v.Type, v.Contents); err != nil {
			return err
		}
	}

	for _, v := range []struct {
		Filename string
		Resource string
		Data     string
	}{
		{"Engine/Extras/Apple File.template", "i'm the resource fork", "this is just the data fork"},
		{"Engine/Extras/Apple File Dst.template", "destination fork", "this is just the data fork"},
	} {
		if err := addAppleFile(pf, cl, filepath.Join(dst.Root(), v.Filename), v.Resource, v.Data); err != nil {
			return err
		}
	}

	if err := pf.SubmitChangelist(cl); err != nil {
		return err
	}

	return nil
}

func addFile(server *p4.P4, cl int64, filename, p4type, contents string) error {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), os.ModePerm); err != nil {
		return fmt.Errorf("error creating dir %s: %w", filepath.Dir(abs), err)
	}
	if err := ioutil.WriteFile(abs, []byte(contents), 0666); err != nil {
		return fmt.Errorf("error writing to %s: %w", abs, err)
	}
	return server.Add([]string{abs}, p4.Type(p4type), p4.Changelist(cl), p4.DoNotIgnore)
}

var doubleResourceHeader = [34]byte{
	0x00, 0x05, 0x16, 0x07, 0x00, 0x02, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x01, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00,
	0x00, 0x26,
}

func addAppleFile(server *p4.P4, cl int64, filename, resource, data string) error {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), os.ModePerm); err != nil {
		return fmt.Errorf("error creating dir %s: %w", filepath.Dir(abs), err)
	}

	// An AppleDouble formatted file is actually made up of two files: one file containing
	// the resource fork, and another containing the data fork.

	// first build the contents of the resource fork file
	b := make([]byte, 0, len(doubleResourceHeader)+4+len(resource))
	b = append(b, doubleResourceHeader[:]...)
	b = binary.BigEndian.AppendUint32(b, uint32(len(resource)))
	b = append(b, []byte(resource)...)

	// then write the resource fork
	path := filepath.Join(filepath.Dir(abs), "%"+filepath.Base(abs))
	if err := ioutil.WriteFile(path, b, 0666); err != nil {
		return fmt.Errorf("error writing Apple Double resource fork to %s: %w", path, err)
	}

	// second write the data to the data fork file
	if err := ioutil.WriteFile(abs, []byte(data), 0666); err != nil {
		return fmt.Errorf("error writing Apple Double data fork to %s: %w", abs, err)
	}

	// AppleDouble files are added by a single call to p4 add (file type must be "apple")
	return server.Add([]string{abs}, p4.Type("apple"), p4.Changelist(cl), p4.DoNotIgnore)
}
